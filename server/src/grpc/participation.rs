use api::pb::participation_service_server::ParticipationService;
use api::pb::{
    CreateParticipationRequest, DeleteParticipationRequest, GetParticipationRequest,
    ListParticipationsRequest, ListParticipationsResponse, Participation,
    UpdateParticipationRequest,
};
use sqlx::PgPool;
use sqlx::FromRow;
use tonic::{Request, Response, Result, Status};

#[derive(Clone, Debug)]
pub struct ParticipationServer {
    pool: PgPool,
}

#[derive(Clone, Debug, FromRow)]
struct ParticipationRow {
    song_id: i32,
    person_id: i64,
    role: String,
}

impl ParticipationServer {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
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

        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            INSERT INTO song_participations (song_id, person_id, role)
            VALUES ($1, $2, $3)
            RETURNING song_id, person_id, role
            "#,
        )
        .bind(participation.song_id as i64)
        .bind(participation.tg_id as i64)
        .bind(participation.role_title)
        .fetch_one(&self.pool)
        .await
        .map_err(map_db_error)?;

        Ok(Response::new(row_to_participation(row)))
    }

    async fn get_participation(
        &self,
        request: Request<GetParticipationRequest>,
    ) -> Result<Response<Participation>, Status> {
        let key = parse_participation_name(&request.into_inner().name)?;

        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            SELECT song_id, person_id, role
            FROM song_participations
            WHERE song_id = $1 AND person_id = $2 AND role = $3
            "#,
        )
        .bind(key.song_id as i64)
        .bind(key.tg_id as i64)
        .bind(key.role_title)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("participation not found"))?;

        Ok(Response::new(row_to_participation(row)))
    }

    async fn list_participations(
        &self,
        request: Request<ListParticipationsRequest>,
    ) -> Result<Response<ListParticipationsResponse>, Status> {
        let limit = sanitize_page_size(request.into_inner().page_size);

        let rows = sqlx::query_as::<_, ParticipationRow>(
            r#"
            SELECT song_id, person_id, role
            FROM song_participations
            ORDER BY song_id, person_id
            LIMIT $1
            "#,
        )
        .bind(limit)
        .fetch_all(&self.pool)
        .await
        .map_err(map_db_error)?;

        let participations = rows.into_iter().map(row_to_participation).collect();
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

        let row = sqlx::query_as::<_, ParticipationRow>(
            r#"
            UPDATE song_participations
            SET role = $1
            WHERE song_id = $2 AND person_id = $3 AND role = $4
            RETURNING song_id, person_id, role
            "#,
        )
        .bind(updated.role_title)
        .bind(participation.song_id as i64)
        .bind(participation.tg_id as i64)
        .bind(participation.role_title)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("participation not found"))?;

        Ok(Response::new(row_to_participation(row)))
    }

    async fn delete_participation(
        &self,
        request: Request<DeleteParticipationRequest>,
    ) -> Result<Response<()>, Status> {
        let key = parse_participation_name(&request.into_inner().name)?;

        let result = sqlx::query(
            r#"
            DELETE FROM song_participations
            WHERE song_id = $1 AND person_id = $2 AND role = $3
            "#,
        )
        .bind(key.song_id as i64)
        .bind(key.tg_id as i64)
        .bind(key.role_title)
        .execute(&self.pool)
        .await
        .map_err(map_db_error)?;

        if result.rows_affected() == 0 {
            return Err(Status::not_found("participation not found"));
        }

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

fn row_to_participation(row: ParticipationRow) -> Participation {
    Participation {
        tg_id: row.person_id as u64,
        song_id: row.song_id as u64,
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
    let paths = mask
        .map(|mask| mask.paths)
        .unwrap_or_else(Vec::new);

    if paths.is_empty() {
        return Ok(updated);
    }

    for path in paths {
        match path.as_str() {
            "role_title" => updated.role_title = participation.role_title.clone(),
            "tg_id" | "song_id" => {
                return Err(Status::invalid_argument(
                    "updating tg_id or song_id is not supported",
                ))
            }
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
    use super::{parse_participation_name, validate_participation};
    use api::pb::Participation;

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
}
