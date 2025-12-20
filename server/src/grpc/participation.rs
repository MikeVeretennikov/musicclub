use std::sync::Arc;

use api::pb::participation_service_server::ParticipationService;
use api::pb::{
    CreateParticipationRequest, DeleteParticipationRequest, GetParticipationRequest,
    ListParticipationsRequest, ListParticipationsResponse, Participation,
    UpdateParticipationRequest,
};
use async_trait::async_trait;
use sqlx::FromRow;
use sqlx::PgPool;
use tonic::{Request, Response, Result, Status};

#[derive(Clone)]
pub struct ParticipationServer {
    store: Arc<dyn ParticipationStore>,
}

#[derive(Clone, Debug, FromRow)]
struct ParticipationRow {
    song_id: i32,
    user_id: i64,
    role: String,
}

#[derive(Clone, Debug)]
pub struct ParticipationRecord {
    pub song_id: u64,
    pub user_id: u64,
    pub role: String,
}

#[derive(Debug)]
pub enum StoreError {
    NotFound,
    Database(String),
}

#[async_trait]
pub trait ParticipationStore: Send + Sync {
    async fn create(&self, record: ParticipationRecord) -> Result<ParticipationRecord, StoreError>;
    async fn get(&self, record: ParticipationRecord) -> Result<ParticipationRecord, StoreError>;
    async fn list(&self, limit: i64) -> Result<Vec<ParticipationRecord>, StoreError>;
    async fn update(
        &self,
        current: ParticipationRecord,
        new_role: String,
    ) -> Result<ParticipationRecord, StoreError>;
    async fn delete(&self, record: ParticipationRecord) -> Result<(), StoreError>;
}

#[derive(Debug)]
struct PostgresParticipationStore {
    pool: PgPool,
}

impl PostgresParticipationStore {
    fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ParticipationStore for PostgresParticipationStore {
    async fn create(&self, record: ParticipationRecord) -> Result<ParticipationRecord, StoreError> {
        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            INSERT INTO song_participations (song_id, users_id, role)
            VALUES ($1, $2, $3)
            RETURNING song_id, users_id, role
            "#,
        )
        .bind(record.song_id as i64)
        .bind(record.user_id as i64)
        .bind(record.role)
        .fetch_one(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?;

        Ok(record_from_row(row))
    }

    async fn get(&self, record: ParticipationRecord) -> Result<ParticipationRecord, StoreError> {
        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            SELECT song_id, user_id, role
            FROM song_participations
            WHERE song_id = $1 AND user_id = $2 AND role = $3
            "#,
        )
        .bind(record.song_id as i64)
        .bind(record.user_id as i64)
        .bind(record.role)
        .fetch_optional(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?
        .ok_or(StoreError::NotFound)?;

        Ok(record_from_row(row))
    }

    async fn list(&self, limit: i64) -> Result<Vec<ParticipationRecord>, StoreError> {
        let rows = sqlx::query_as::<_, ParticipationRow>(
            r#"
            SELECT song_id, user_id, role
            FROM song_participations
            ORDER BY song_id, user_id
            LIMIT $1
            "#,
        )
        .bind(limit)
        .fetch_all(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?;

        Ok(rows.into_iter().map(record_from_row).collect())
    }

    async fn update(
        &self,
        current: ParticipationRecord,
        new_role: String,
    ) -> Result<ParticipationRecord, StoreError> {
        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            UPDATE song_participations
            SET role = $1
            WHERE song_id = $2 AND user_id = $3 AND role = $4
            RETURNING song_id, user_id, role
            "#,
        )
        .bind(new_role)
        .bind(current.song_id as i64)
        .bind(current.user_id as i64)
        .bind(current.role)
        .fetch_optional(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?
        .ok_or(StoreError::NotFound)?;

        Ok(record_from_row(row))
    }

    async fn delete(&self, record: ParticipationRecord) -> Result<(), StoreError> {
        let result = sqlx::query(
            r#"
            DELETE FROM song_participations
            WHERE song_id = $1 AND user_id = $2 AND role = $3
            "#,
        )
        .bind(record.song_id as i64)
        .bind(record.user_id as i64)
        .bind(record.role)
        .execute(&self.pool)
        .await
        .map_err(|err| StoreError::Database(err.to_string()))?;

        if result.rows_affected() == 0 {
            return Err(StoreError::NotFound);
        }

        Ok(())
    }
}

impl ParticipationServer {
    pub fn new(pool: PgPool) -> Self {
        Self {
            store: Arc::new(PostgresParticipationStore::new(pool)),
        }
    }

    #[allow(dead_code)]
    pub fn with_store(store: Arc<dyn ParticipationStore>) -> Self {
        Self { store }
    }
}

#[tonic::async_trait]
impl ParticipationService for ParticipationServer {
    async fn create_participation(
        &self,
        request: Request<CreateParticipationRequest>,
    ) -> Result<Response<Participation>, Status> {
        let participation = request
            .into_inner()
            .participation
            .ok_or_else(|| Status::invalid_argument("participation payload is required"))?;

        validate_participation(&participation)?;

        let record = ParticipationRecord {
            song_id: participation.song_id,
            user_id: participation.tg_id,
            role: participation.role_title,
        };
        let record = self.store.create(record).await.map_err(map_store_error)?;
        Ok(Response::new(record_to_participation(record)))
    }

    async fn get_participation(
        &self,
        request: Request<GetParticipationRequest>,
    ) -> Result<Response<Participation>, Status> {
        let key = parse_participation_name(&request.into_inner().name)?;

        let record = ParticipationRecord {
            song_id: key.song_id,
            user_id: key.tg_id,
            role: key.role_title,
        };
        let record = self.store.get(record).await.map_err(map_store_error)?;
        Ok(Response::new(record_to_participation(record)))
    }

    async fn list_participations(
        &self,
        request: Request<ListParticipationsRequest>,
    ) -> Result<Response<ListParticipationsResponse>, Status> {
        let limit = sanitize_page_size(request.into_inner().page_size);

        let rows = self.store.list(limit).await.map_err(map_store_error)?;
        let participations = rows.into_iter().map(record_to_participation).collect();
        Ok(Response::new(ListParticipationsResponse {
            participations,
            next_page_token: String::new(),
        }))
    }

    async fn update_participation(
        &self,
        request: Request<UpdateParticipationRequest>,
    ) -> Result<Response<Participation>, Status> {
        let request = request.into_inner();
        let participation = request
            .participation
            .ok_or_else(|| Status::invalid_argument("participation payload is required"))?;
        validate_participation(&participation)?;

        let updated = apply_participation_update_mask(&participation, request.update_mask)?;
        let current = ParticipationRecord {
            song_id: participation.song_id,
            user_id: participation.tg_id,
            role: participation.role_title,
        };
        let record = self
            .store
            .update(current, updated.role_title)
            .await
            .map_err(map_store_error)?;
        Ok(Response::new(record_to_participation(record)))
    }

    async fn delete_participation(
        &self,
        request: Request<DeleteParticipationRequest>,
    ) -> Result<Response<()>, Status> {
        let key = parse_participation_name(&request.into_inner().name)?;

        let record = ParticipationRecord {
            song_id: key.song_id,
            user_id: key.tg_id,
            role: key.role_title,
        };
        self.store.delete(record).await.map_err(map_store_error)?;
        Ok(Response::new(()))
    }
}

fn sanitize_page_size(page_size: i32) -> i64 {
    let size = if page_size <= 0 { 100 } else { page_size };
    i64::from(size.min(500))
}

fn validate_participation(participation: &Participation) -> Result<(), Status> {
    if participation.tg_id == 0 {
        return Err(Status::invalid_argument("tg_id is required"));
    }
    if participation.song_id == 0 {
        return Err(Status::invalid_argument("song_id is required"));
    }
    if participation.role_title.trim().is_empty() {
        return Err(Status::invalid_argument("role_title is required"));
    }
    Ok(())
}

fn record_from_row(row: ParticipationRow) -> ParticipationRecord {
    ParticipationRecord {
        song_id: row.song_id as u64,
        user_id: row.user_id as u64,
        role: row.role,
    }
}

fn record_to_participation(row: ParticipationRecord) -> Participation {
    Participation {
        tg_id: row.user_id,
        song_id: row.song_id,
        role_title: row.role,
    }
}

#[derive(Debug)]
struct ParticipationKey {
    song_id: u64,
    tg_id: u64,
    role_title: String,
}

fn parse_participation_name(name: &str) -> Result<ParticipationKey, Status> {
    let mut parts = name.splitn(3, ':');
    let song_id = parts
        .next()
        .ok_or_else(|| Status::invalid_argument("invalid participation name"))?;
    let tg_id = parts
        .next()
        .ok_or_else(|| Status::invalid_argument("invalid participation name"))?;
    let role_title = parts
        .next()
        .ok_or_else(|| Status::invalid_argument("invalid participation name"))?;

    let song_id = song_id
        .trim()
        .parse::<u64>()
        .map_err(|_| Status::invalid_argument("invalid participation name"))?;
    let tg_id = tg_id
        .trim()
        .parse::<u64>()
        .map_err(|_| Status::invalid_argument("invalid participation name"))?;

    if song_id == 0 || tg_id == 0 || role_title.trim().is_empty() {
        return Err(Status::invalid_argument("invalid participation name"));
    }

    Ok(ParticipationKey {
        song_id,
        tg_id,
        role_title: role_title.to_string(),
    })
}

fn apply_participation_update_mask(
    participation: &Participation,
    mask: Option<prost_types::FieldMask>,
) -> Result<Participation, Status> {
    let mut updated = participation.clone();
    let paths = mask.map(|mask| mask.paths).unwrap_or_else(Vec::new);

    if paths.is_empty() {
        return Ok(updated);
    }

    for path in paths {
        match path.as_str() {
            "role_title" => updated.role_title = participation.role_title.clone(),
            "tg_id" | "song_id" => {
                return Err(Status::invalid_argument(
                    "updating tg_id or song_id is not supported",
                ));
            }
            _ => return Err(Status::invalid_argument("unsupported update_mask path")),
        }
    }

    Ok(updated)
}

fn map_store_error(err: StoreError) -> Status {
    match err {
        StoreError::NotFound => Status::not_found("participation not found"),
        StoreError::Database(message) => Status::internal(format!("database error: {message}")),
    }
}

#[cfg(test)]
mod tests {
    use super::{
        ParticipationRecord, ParticipationServer, ParticipationStore, StoreError,
        parse_participation_name, validate_participation,
    };
    use api::pb::Participation;
    use api::pb::participation_service_client::ParticipationServiceClient;
    use api::pb::participation_service_server::ParticipationServiceServer;
    use api::pb::{
        CreateParticipationRequest, DeleteParticipationRequest, GetParticipationRequest,
        ListParticipationsRequest, UpdateParticipationRequest,
    };
    use async_trait::async_trait;
    use sqlx::{PgPool, postgres::PgPoolOptions};
    use std::collections::HashMap;
    use std::net::SocketAddr;
    use std::sync::Arc;
    use tokio::sync::Mutex;
    use tokio_stream::wrappers::TcpListenerStream;
    use tonic::transport::Channel;
    use tonic::{Request, transport::Server};

    #[test]
    fn parse_participation_name_accepts_triplet() {
        let key = parse_participation_name("10:20:Drums").expect("key");
        assert_eq!(key.song_id, 10);
        assert_eq!(key.tg_id, 20);
        assert_eq!(key.role_title, "Drums");
    }

    #[test]
    fn validate_participation_requires_fields() {
        let bad = Participation {
            tg_id: 0,
            song_id: 0,
            role_title: "".to_string(),
        };
        assert!(validate_participation(&bad).is_err());
    }

    #[derive(Debug)]
    struct MockParticipationStore {
        data: Mutex<HashMap<(u64, u64, String), ParticipationRecord>>,
        _pool: PgPool,
    }

    impl MockParticipationStore {
        fn new() -> Self {
            Self {
                data: Mutex::new(HashMap::new()),
                _pool: PgPoolOptions::new()
                    .connect_lazy("postgres://postgres:postgres@localhost/postgres")
                    .expect("stub pool"),
            }
        }
    }

    #[async_trait]
    impl ParticipationStore for MockParticipationStore {
        async fn create(
            &self,
            record: ParticipationRecord,
        ) -> Result<ParticipationRecord, StoreError> {
            let key = (record.song_id, record.user_id, record.role.clone());
            self.data.lock().await.insert(key, record.clone());
            Ok(record)
        }

        async fn get(
            &self,
            record: ParticipationRecord,
        ) -> Result<ParticipationRecord, StoreError> {
            let key = (record.song_id, record.user_id, record.role.clone());
            self.data
                .lock()
                .await
                .get(&key)
                .cloned()
                .ok_or(StoreError::NotFound)
        }

        async fn list(&self, limit: i64) -> Result<Vec<ParticipationRecord>, StoreError> {
            let mut values: Vec<_> = self.data.lock().await.values().cloned().collect();
            values.sort_by_key(|rec| (rec.song_id, rec.user_id, rec.role.clone()));
            values.truncate(limit as usize);
            Ok(values)
        }

        async fn update(
            &self,
            current: ParticipationRecord,
            new_role: String,
        ) -> Result<ParticipationRecord, StoreError> {
            let mut data = self.data.lock().await;
            let key = (current.song_id, current.user_id, current.role.clone());
            if data.remove(&key).is_none() {
                return Err(StoreError::NotFound);
            }
            let updated = ParticipationRecord {
                song_id: current.song_id,
                user_id: current.user_id,
                role: new_role,
            };
            let new_key = (updated.song_id, updated.user_id, updated.role.clone());
            data.insert(new_key, updated.clone());
            Ok(updated)
        }

        async fn delete(&self, record: ParticipationRecord) -> Result<(), StoreError> {
            let key = (record.song_id, record.user_id, record.role);
            if self.data.lock().await.remove(&key).is_none() {
                return Err(StoreError::NotFound);
            }
            Ok(())
        }
    }

    async fn start_server(
        store: Arc<dyn ParticipationStore>,
    ) -> Option<(SocketAddr, tokio::task::JoinHandle<()>)> {
        let addr: SocketAddr = "127.0.0.1:0".parse().expect("addr");
        let listener = match tokio::net::TcpListener::bind(&addr).await {
            Ok(listener) => listener,
            Err(err) if err.kind() == std::io::ErrorKind::PermissionDenied => return None,
            Err(err) => panic!("bind failed: {err}"),
        };
        let addr = listener.local_addr().expect("local addr");
        let service = ParticipationServiceServer::new(ParticipationServer::with_store(store));

        let handle = tokio::spawn(async move {
            Server::builder()
                .add_service(service)
                .serve_with_incoming(TcpListenerStream::new(listener))
                .await
                .expect("grpc server failed");
        });

        tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;
        Some((addr, handle))
    }

    async fn create_client(addr: SocketAddr) -> ParticipationServiceClient<Channel> {
        let endpoint = format!("http://{}:{}", addr.ip(), addr.port());
        ParticipationServiceClient::connect(endpoint)
            .await
            .expect("connect")
    }

    #[tokio::test]
    async fn e2e_participation_crud() {
        let store = Arc::new(MockParticipationStore::new());
        let Some((addr, _handle)) = start_server(store).await else {
            eprintln!("skipping e2e_participation_crud: tcp bind not permitted");
            return;
        };
        let mut client = create_client(addr).await;

        let create = CreateParticipationRequest {
            parent: String::new(),
            participation_id: String::new(),
            participation: Some(Participation {
                tg_id: 1,
                song_id: 2,
                role_title: "Guitar".to_string(),
            }),
        };
        let created = client
            .create_participation(Request::new(create))
            .await
            .expect("create")
            .into_inner();
        assert_eq!(created.role_title, "Guitar");

        let fetched = client
            .get_participation(Request::new(GetParticipationRequest {
                name: "2:1:Guitar".to_string(),
            }))
            .await
            .expect("get")
            .into_inner();
        assert_eq!(fetched.song_id, 2);

        let list = client
            .list_participations(Request::new(ListParticipationsRequest {
                parent: String::new(),
                page_size: 10,
                page_token: String::new(),
            }))
            .await
            .expect("list")
            .into_inner();
        assert_eq!(list.participations.len(), 1);

        let updated = client
            .update_participation(Request::new(UpdateParticipationRequest {
                participation: Some(Participation {
                    tg_id: 1,
                    song_id: 2,
                    role_title: "Guitar".to_string(),
                }),
                update_mask: Some(prost_types::FieldMask {
                    paths: vec!["role_title".to_string()],
                }),
            }))
            .await
            .expect("update")
            .into_inner();
        assert_eq!(updated.role_title, "Guitar");

        client
            .delete_participation(Request::new(DeleteParticipationRequest {
                name: "2:1:Guitar".to_string(),
            }))
            .await
            .expect("delete");
    }
}
