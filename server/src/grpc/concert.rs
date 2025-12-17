use api::pb::concert_service_server::ConcertService;
use api::pb::{
    Concert, CreateConcertRequest, DeleteConcertRequest, GetConcertRequest, ListConcertsRequest,
    ListConcertsResponse, UpdateConcertRequest,
};
use chrono::{DateTime, NaiveDate, Utc};
use prost_types::Timestamp;
use sqlx::PgPool;
use sqlx::FromRow;
use tonic::{Request, Response, Result, Status};

#[derive(Clone, Debug)]
pub struct ConcertServer {
    pool: PgPool,
}

#[derive(Clone, Debug, FromRow)]
struct ConcertRow {
    id: i32,
    name: String,
    date: Option<NaiveDate>,
}

impl ConcertServer {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[tonic::async_trait]
impl ConcertService for ConcertServer {
    async fn create_concert(
        &self,
        request: Request<CreateConcertRequest>,
    ) -> Result<Response<Concert>, Status> {
        let concert = request.into_inner().concert.ok_or_else(|| {
            Status::invalid_argument("create_concert requires concert payload")
        })?;
        if concert.name.trim().is_empty() {
            return Err(Status::invalid_argument("concert name is required"));
        }

        let row = match date_from_timestamp(concert.date) {
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
            .map_err(map_db_error)?,
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
            .map_err(map_db_error)?,
        };

        Ok(Response::new(row_to_concert(row)))
    }

    async fn get_concert(
        &self,
        request: Request<GetConcertRequest>,
    ) -> Result<Response<Concert>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        let row = sqlx::query_as::<_, ConcertRow>(
            r#"
            SELECT id, name, date
            FROM concerts
            WHERE id = $1
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("concert not found"))?;

        Ok(Response::new(row_to_concert(row)))
    }

    async fn list_concerts(
        &self,
        request: Request<ListConcertsRequest>,
    ) -> Result<Response<ListConcertsResponse>, Status> {
        let limit = sanitize_page_size(request.into_inner().page_size);

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
        .map_err(map_db_error)?;

        let concerts = rows.into_iter().map(row_to_concert).collect();
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

        let existing = sqlx::query_as::<_, ConcertRow>(
            r#"
            SELECT id, name, date
            FROM concerts
            WHERE id = $1
            "#,
        )
        .bind(concert.id as i64)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("concert not found"))?;

        let updated = apply_concert_update_mask(&existing, &concert, request.update_mask)?;
        if updated.name.trim().is_empty() {
            return Err(Status::invalid_argument("concert name is required"));
        }

        let row = sqlx::query_as::<_, ConcertRow>(
            r#"
            UPDATE concerts
            SET name = $1, date = $2
            WHERE id = $3
            RETURNING id, name, date
            "#,
        )
        .bind(updated.name)
        .bind(date_from_timestamp(updated.date))
        .bind(concert.id as i64)
        .fetch_one(&self.pool)
        .await
        .map_err(map_db_error)?;

        Ok(Response::new(row_to_concert(row)))
    }

    async fn delete_concert(
        &self,
        request: Request<DeleteConcertRequest>,
    ) -> Result<Response<()>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        let result = sqlx::query("DELETE FROM concerts WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await
            .map_err(map_db_error)?;

        if result.rows_affected() == 0 {
            return Err(Status::not_found("concert not found"));
        }

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

fn row_to_concert(row: ConcertRow) -> Concert {
    Concert {
        id: row.id as u64,
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
    existing: &ConcertRow,
    incoming: &Concert,
    mask: Option<prost_types::FieldMask>,
) -> Result<Concert, Status> {
    let mut updated = row_to_concert(existing.clone());
    let paths = mask
        .map(|mask| mask.paths)
        .unwrap_or_else(Vec::new);

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

fn map_db_error(err: sqlx::Error) -> Status {
    Status::internal(format!("database error: {err}"))
}

#[cfg(test)]
mod tests {
    use super::{apply_concert_update_mask, date_from_timestamp, row_to_concert, ConcertRow};
    use api::pb::Concert;
    use chrono::NaiveDate;
    use prost_types::Timestamp;

    #[test]
    fn date_roundtrip_works() {
        let date = NaiveDate::from_ymd_opt(2025, 1, 2).expect("date");
        let timestamp = super::timestamp_from_date(date).expect("timestamp");
        let parsed = date_from_timestamp(Some(timestamp)).expect("parsed");
        assert_eq!(parsed, date);
    }

    #[test]
    fn update_mask_keeps_existing_fields() {
        let existing = ConcertRow {
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
        assert_eq!(updated.date, row_to_concert(existing).date);
    }
}
