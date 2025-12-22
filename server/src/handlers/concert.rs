use std::sync::Arc;

use api::pb::concert_service_server::ConcertService;
use api::pb::{
    Concert, CreateConcertRequest, DeleteConcertRequest, GetConcertRequest, ListConcertsRequest,
    ListConcertsResponse, UpdateConcertRequest,
};
use async_trait::async_trait;
use chrono::{DateTime, NaiveDate, Utc};
use prost_types::Timestamp;
use sqlx::FromRow;
use sqlx::PgPool;
use tonic::{Request, Response, Result, Status};

#[derive(Clone)]
pub struct ConcertServer {
    store: Arc<dyn ConcertStore>,
}

#[derive(Clone, Debug, FromRow)]
struct ConcertRow {
    id: i32,
    name: String,
    date: Option<NaiveDate>,
}

#[derive(Clone, Debug)]
pub struct ConcertRecord {
    pub id: u64,
    pub name: String,
    pub date: Option<NaiveDate>,
}

#[derive(Debug)]
pub enum StoreError {
    NotFound,
    Database(String),
}

#[async_trait]
pub trait ConcertStore: Send + Sync {
    async fn create(&self, concert: ConcertRecord) -> Result<ConcertRecord, StoreError>;
    async fn get(&self, id: u64) -> Result<ConcertRecord, StoreError>;
    async fn list(&self, limit: i64) -> Result<Vec<ConcertRecord>, StoreError>;
    async fn update(&self, concert: ConcertRecord) -> Result<ConcertRecord, StoreError>;
    async fn delete(&self, id: u64) -> Result<(), StoreError>;
}

#[derive(Debug)]
struct PostgresConcertStore {
    pool: PgPool,
}

impl PostgresConcertStore {
    fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ConcertStore for PostgresConcertStore {
    async fn create(&self, concert: ConcertRecord) -> Result<ConcertRecord, StoreError> {
        let row = match concert.date {
            Some(date) => sqlx::query_as::<_, ConcertRow>(
                r#"
                INSERT INTO concerts (name, date)
                VALUES ($1, $2)
                RETURNING id, name, date
                "#,
            )
            .bind(concert.name)
            .bind(date)
            .fetch_one(&self.pool)
            .await
            .map_err(|err| StoreError::Database(err.to_string()))?,
            None => sqlx::query_as::<_, ConcertRow>(
                r#"
                INSERT INTO concerts (name)
                VALUES ($1)
                RETURNING id, name, date
                "#,
            )
            .bind(concert.name)
            .fetch_one(&self.pool)
            .await
            .map_err(|err| StoreError::Database(err.to_string()))?,
        };

        Ok(record_from_row(row))
    }

    async fn get(&self, id: u64) -> Result<ConcertRecord, StoreError> {
        let row = sqlx::query_as::<_, ConcertRow>(
            r#"
            SELECT id, name, date
            FROM concerts
            WHERE id = $1
            "#,
        )
        .bind(id as i64)
        .fetch_optional(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?
        .ok_or(StoreError::NotFound)?;

        Ok(record_from_row(row))
    }

    async fn list(&self, limit: i64) -> Result<Vec<ConcertRecord>, StoreError> {
        let rows = sqlx::query_as::<_, ConcertRow>(
            r#"
            SELECT id, name, date
            FROM concerts
            ORDER BY id
            LIMIT $1
            "#,
        )
        .bind(limit)
        .fetch_all(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?;

        Ok(rows.into_iter().map(record_from_row).collect())
    }

    async fn update(&self, concert: ConcertRecord) -> Result<ConcertRecord, StoreError> {
        let row = sqlx::query_as::<_, ConcertRow>(
            r#"
            UPDATE concerts
            SET name = $1, date = $2
            WHERE id = $3
            RETURNING id, name, date
            "#,
        )
        .bind(concert.name)
        .bind(concert.date)
        .bind(concert.id as i64)
        .fetch_optional(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?
        .ok_or(StoreError::NotFound)?;

        Ok(record_from_row(row))
    }

    async fn delete(&self, id: u64) -> Result<(), StoreError> {
        let result = sqlx::query("DELETE FROM concerts WHERE id = $1")
            .bind(id as i64)
            .execute(&self.pool)
            .await
            .map_err(|err| StoreError::Database(err.to_string()))?;

        if result.rows_affected() == 0 {
            return Err(StoreError::NotFound);
        }

        Ok(())
    }
}

impl ConcertServer {
    pub fn new(pool: PgPool) -> Self {
        Self {
            store: Arc::new(PostgresConcertStore::new(pool)),
        }
    }

    #[allow(dead_code)]
    pub fn with_store(store: Arc<dyn ConcertStore>) -> Self {
        Self { store }
    }
}

#[tonic::async_trait]
impl ConcertService for ConcertServer {
    async fn create_concert(
        &self,
        request: Request<CreateConcertRequest>,
    ) -> Result<Response<Concert>, Status> {
        let concert = request
            .into_inner()
            .concert
            .ok_or_else(|| Status::invalid_argument("create_concert requires concert payload"))?;
        if concert.name.trim().is_empty() {
            return Err(Status::invalid_argument("concert name is required"));
        }

        let record = ConcertRecord {
            id: 0,
            name: concert.name,
            date: date_from_timestamp(concert.date),
        };
        let record = self.store.create(record).await.map_err(map_store_error)?;
        Ok(Response::new(record_to_concert(record)))
    }

    async fn get_concert(
        &self,
        request: Request<GetConcertRequest>,
    ) -> Result<Response<Concert>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        let record = self.store.get(id as u64).await.map_err(map_store_error)?;
        Ok(Response::new(record_to_concert(record)))
    }

    async fn list_concerts(
        &self,
        request: Request<ListConcertsRequest>,
    ) -> Result<Response<ListConcertsResponse>, Status> {
        let limit = sanitize_page_size(request.into_inner().page_size);

        let rows = self.store.list(limit).await.map_err(map_store_error)?;
        let concerts = rows.into_iter().map(record_to_concert).collect();
        Ok(Response::new(ListConcertsResponse {
            concerts,
            next_page_token: String::new(),
        }))
    }

    async fn update_concert(
        &self,
        request: Request<UpdateConcertRequest>,
    ) -> Result<Response<Concert>, Status> {
        let request = request.into_inner();
        let concert = request
            .concert
            .ok_or_else(|| Status::invalid_argument("update_concert requires concert payload"))?;
        if concert.id == 0 {
            return Err(Status::invalid_argument("concert id is required"));
        }

        let existing = self.store.get(concert.id).await.map_err(map_store_error)?;
        let updated = apply_concert_update_mask(&existing, &concert, request.update_mask)?;
        if updated.name.trim().is_empty() {
            return Err(Status::invalid_argument("concert name is required"));
        }

        let record = ConcertRecord {
            id: concert.id,
            name: updated.name,
            date: date_from_timestamp(updated.date),
        };
        let record = self.store.update(record).await.map_err(map_store_error)?;
        Ok(Response::new(record_to_concert(record)))
    }

    async fn delete_concert(
        &self,
        request: Request<DeleteConcertRequest>,
    ) -> Result<Response<()>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        self.store
            .delete(id as u64)
            .await
            .map_err(map_store_error)?;
        Ok(Response::new(()))
    }
}

fn sanitize_page_size(page_size: i32) -> i64 {
    let size = if page_size <= 0 { 100 } else { page_size };
    i64::from(size.min(500))
}

fn parse_id(name: &str) -> Result<i64, Status> {
    name.trim()
        .parse::<i64>()
        .map_err(|_| Status::invalid_argument("invalid id"))
        .and_then(|id| {
            if id <= 0 {
                Err(Status::invalid_argument("invalid id"))
            } else {
                Ok(id)
            }
        })
}

fn record_from_row(row: ConcertRow) -> ConcertRecord {
    ConcertRecord {
        id: row.id as u64,
        name: row.name,
        date: row.date,
    }
}

fn record_to_concert(row: ConcertRecord) -> Concert {
    Concert {
        id: row.id,
        name: row.name,
        date: row.date.and_then(timestamp_from_date),
    }
}

fn timestamp_from_date(date: NaiveDate) -> Option<Timestamp> {
    let datetime = date.and_hms_opt(0, 0, 0)?.and_utc();
    Some(Timestamp {
        seconds: datetime.timestamp(),
        nanos: 0,
    })
}

fn date_from_timestamp(timestamp: Option<Timestamp>) -> Option<NaiveDate> {
    let timestamp = timestamp?;
    let nanos = u32::try_from(timestamp.nanos).ok()?;
    DateTime::<Utc>::from_timestamp(timestamp.seconds, nanos).map(|dt| dt.date_naive())
}

fn apply_concert_update_mask(
    existing: &ConcertRecord,
    incoming: &Concert,
    mask: Option<prost_types::FieldMask>,
) -> Result<Concert, Status> {
    let mut updated = record_to_concert(existing.clone());
    let paths = mask.map(|mask| mask.paths).unwrap_or_else(Vec::new);

    if paths.is_empty() {
        updated.name = incoming.name.clone();
        updated.date = incoming.date.clone();
        return Ok(updated);
    }

    for path in paths {
        match path.as_str() {
            "name" => updated.name = incoming.name.clone(),
            "date" => updated.date = incoming.date.clone(),
            _ => return Err(Status::invalid_argument("unsupported update_mask path")),
        }
    }

    Ok(updated)
}

fn map_store_error(err: StoreError) -> Status {
    match err {
        StoreError::NotFound => Status::not_found("concert not found"),
        StoreError::Database(message) => Status::internal(format!("database error: {message}")),
    }
}

#[cfg(test)]
mod tests {
    use super::{
        ConcertRecord, ConcertServer, ConcertStore, StoreError, apply_concert_update_mask,
        date_from_timestamp, record_to_concert,
    };
    use api::pb::Concert;
    use api::pb::auth_service_client::AuthServiceClient;
    use api::pb::auth_service_server::AuthServiceServer;
    use api::pb::concert_service_client::ConcertServiceClient;
    use api::pb::concert_service_server::ConcertServiceServer;
    use api::pb::{
        CreateConcertRequest, GetConcertRequest, ListConcertsRequest, UpdateConcertRequest,
    };
    use async_trait::async_trait;
    use chrono::NaiveDate;
    use prost_types::Timestamp;
    use sqlx::{PgPool, postgres::PgPoolOptions};
    use std::collections::{HashMap, HashSet};
    use std::net::SocketAddr;
    use std::sync::Arc;
    use std::sync::atomic::{AtomicU64, Ordering};
    use tokio::sync::Mutex;
    use tokio_stream::wrappers::TcpListenerStream;
    use tonic::transport::Channel;
    use tonic::{Request, transport::Server};
    use tonic_middleware::{MiddlewareLayer, RequestInterceptorLayer};

    use crate::handlers::auth::AuthServer;
    use crate::middleware::{AdminOnlyMiddleware, AuthInterceptor};

    #[test]
    fn date_roundtrip_works() {
        let date = NaiveDate::from_ymd_opt(2025, 1, 2).expect("date");
        let timestamp = super::timestamp_from_date(date).expect("timestamp");
        let parsed = date_from_timestamp(Some(timestamp)).expect("parsed");
        assert_eq!(parsed, date);
    }

    #[test]
    fn update_mask_keeps_existing_fields() {
        let existing = ConcertRecord {
            id: 5,
            name: "Old".to_string(),
            date: Some(NaiveDate::from_ymd_opt(2024, 5, 1).expect("date")),
        };
        let incoming = Concert {
            id: 5,
            name: "New".to_string(),
            date: Some(Timestamp {
                seconds: 0,
                nanos: 0,
            }),
        };
        let mask = prost_types::FieldMask {
            paths: vec!["name".to_string()],
        };

        let updated = apply_concert_update_mask(&existing, &incoming, Some(mask)).expect("update");
        assert_eq!(updated.name, "New");
        assert_eq!(updated.date, record_to_concert(existing).date);
    }

    #[derive(Debug)]
    struct MockConcertStore {
        data: Mutex<HashMap<u64, ConcertRecord>>,
        next_id: AtomicU64,
        _pool: PgPool,
    }

    impl MockConcertStore {
        fn new() -> Self {
            Self {
                data: Mutex::new(HashMap::new()),
                next_id: AtomicU64::new(1),
                _pool: PgPoolOptions::new()
                    .connect_lazy("postgres://postgres:postgres@localhost/postgres")
                    .expect("stub pool"),
            }
        }
    }

    #[async_trait]
    impl ConcertStore for MockConcertStore {
        async fn create(&self, mut concert: ConcertRecord) -> Result<ConcertRecord, StoreError> {
            let id = self.next_id.fetch_add(1, Ordering::SeqCst);
            concert.id = id;
            self.data.lock().await.insert(id, concert.clone());
            Ok(concert)
        }

        async fn get(&self, id: u64) -> Result<ConcertRecord, StoreError> {
            self.data
                .lock()
                .await
                .get(&id)
                .cloned()
                .ok_or(StoreError::NotFound)
        }

        async fn list(&self, limit: i64) -> Result<Vec<ConcertRecord>, StoreError> {
            let mut values: Vec<_> = self.data.lock().await.values().cloned().collect();
            values.sort_by_key(|concert| concert.id);
            values.truncate(limit as usize);
            Ok(values)
        }

        async fn update(&self, concert: ConcertRecord) -> Result<ConcertRecord, StoreError> {
            let mut data = self.data.lock().await;
            if !data.contains_key(&concert.id) {
                return Err(StoreError::NotFound);
            }
            data.insert(concert.id, concert.clone());
            Ok(concert)
        }

        async fn delete(&self, id: u64) -> Result<(), StoreError> {
            let mut data = self.data.lock().await;
            if data.remove(&id).is_none() {
                return Err(StoreError::NotFound);
            }
            Ok(())
        }
    }

    async fn start_server(
        store: Arc<dyn ConcertStore>,
        admin_ids: HashSet<u64>,
    ) -> Option<(SocketAddr, tokio::task::JoinHandle<()>)> {
        let addr: SocketAddr = "127.0.0.1:0".parse().expect("addr");
        let listener = match tokio::net::TcpListener::bind(&addr).await {
            Ok(listener) => listener,
            Err(err) if err.kind() == std::io::ErrorKind::PermissionDenied => return None,
            Err(err) => panic!("bind failed: {err}"),
        };
        let addr = listener.local_addr().expect("local addr");
        let secret = b"secret";
        let auth = AuthServer::new(
            secret,
            admin_ids.clone(),
            std::time::Duration::from_secs(3600),
        );
        let interceptor = AuthInterceptor::new(secret);
        let admin_middleware = AdminOnlyMiddleware::new(admin_ids);

        let concert_service = ConcertServiceServer::new(ConcertServer::with_store(store));

        let handle = tokio::spawn(async move {
            Server::builder()
                .layer(RequestInterceptorLayer::new(interceptor))
                .layer(MiddlewareLayer::new(admin_middleware))
                .add_service(AuthServiceServer::new(auth))
                .add_service(concert_service)
                .serve_with_incoming(TcpListenerStream::new(listener))
                .await
                .expect("grpc server failed");
        });

        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;
        Some((addr, handle))
    }

    async fn create_concert_client(addr: SocketAddr) -> ConcertServiceClient<Channel> {
        let endpoint = format!("http://{}:{}", addr.ip(), addr.port());
        ConcertServiceClient::connect(endpoint)
            .await
            .expect("connect")
    }

    async fn create_auth_client(addr: SocketAddr) -> AuthServiceClient<Channel> {
        let endpoint = format!("http://{}:{}", addr.ip(), addr.port());
        AuthServiceClient::connect(endpoint).await.expect("connect")
    }

    #[tokio::test]
    async fn e2e_concert_crud_with_admin_guard() {
        let store = Arc::new(MockConcertStore::new());
        let admin_ids = HashSet::from([42_u64]);
        let Some((addr, _handle)) = start_server(store, admin_ids).await else {
            eprintln!("skipping e2e_concert_crud_with_admin_guard: tcp bind not permitted");
            return;
        };

        let mut auth_client = create_auth_client(addr).await;
        let admin_token = auth_client
            .login_tg(Request::new(api::pb::TgLogin {
                tg_id: 42,
                init_data: "tg_id=42".to_string(),
            }))
            .await
            .expect("login admin")
            .into_inner()
            .token;
        let user_token = auth_client
            .login_tg(Request::new(api::pb::TgLogin {
                tg_id: 7,
                init_data: "tg_id=7".to_string(),
            }))
            .await
            .expect("login user")
            .into_inner()
            .token;

        let mut client = create_concert_client(addr).await;
        let mut req = Request::new(CreateConcertRequest {
            parent: String::new(),
            concert_id: String::new(),
            concert: Some(Concert {
                id: 0,
                name: "Concert".to_string(),
                date: None,
            }),
        });
        req.metadata_mut().insert(
            "authorization",
            format!("Bearer {}", user_token).parse().unwrap(),
        );
        let err = client.create_concert(req).await.unwrap_err();
        assert_eq!(err.code(), tonic::Code::PermissionDenied);

        let mut admin_req = Request::new(CreateConcertRequest {
            parent: String::new(),
            concert_id: String::new(),
            concert: Some(Concert {
                id: 0,
                name: "Concert".to_string(),
                date: None,
            }),
        });
        admin_req.metadata_mut().insert(
            "authorization",
            format!("Bearer {}", admin_token).parse().unwrap(),
        );
        let created = client
            .create_concert(admin_req)
            .await
            .expect("create")
            .into_inner();

        let fetched = client
            .get_concert(Request::new(GetConcertRequest {
                name: created.id.to_string(),
            }))
            .await
            .expect("get")
            .into_inner();
        assert_eq!(fetched.name, "Concert");

        let list = client
            .list_concerts(Request::new(ListConcertsRequest {
                parent: String::new(),
                page_size: 10,
                page_token: String::new(),
            }))
            .await
            .expect("list")
            .into_inner();
        assert_eq!(list.concerts.len(), 1);

        let updated = client
            .update_concert(Request::new(UpdateConcertRequest {
                concert: Some(Concert {
                    id: created.id,
                    name: "Updated".to_string(),
                    date: None,
                }),
                update_mask: None,
            }))
            .await
            .expect("update")
            .into_inner();
        assert_eq!(updated.name, "Updated");
    }
}
