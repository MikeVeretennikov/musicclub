use api::pb::song_service_server::SongService;
use api::pb::{
    CreateSongRequest, DeleteSongRequest, GetSongRequest, ListSongsRequest, ListSongsResponse,
    Song, UpdateSongRequest,
};
use sqlx::PgPool;
use sqlx::FromRow;
use tonic::{Request, Response, Result, Status};

#[derive(Clone, Debug)]
pub struct SongServer {
    pool: PgPool,
}

#[derive(Clone, Debug, FromRow)]
struct SongRow {
    id: i32,
    title: String,
    description: Option<String>,
    link: Option<String>,
}

impl SongServer {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[tonic::async_trait]
impl SongService for SongServer {
    async fn create_song(
        &self,
        request: Request<CreateSongRequest>,
    ) -> Result<Response<Song>, Status> {
        let song = request.into_inner().song.ok_or_else(|| {
            Status::invalid_argument("create_song requires song payload")
        })?;

        if song.title.trim().is_empty() {
            return Err(Status::invalid_argument("song title is required"));
        }

        let row = sqlx::query_as::<_, SongRow>(
            r#"
            INSERT INTO songs (title, description, link)
            VALUES ($1, $2, $3)
            RETURNING id, title, description, link
            "#,
        )
        .bind(song.title)
        .bind(empty_to_none(song.description))
        .bind(empty_to_none(song.link))
        .fetch_one(&self.pool)
        .await
        .map_err(map_db_error)?;

        Ok(Response::new(row_to_song(row)))
    }

    async fn get_song(&self, request: Request<GetSongRequest>) -> Result<Response<Song>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        let row = sqlx::query_as::<_, SongRow>(
            r#"
            SELECT id, title, description, link
            FROM songs
            WHERE id = $1
            "#,
        )
        .bind(id)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("song not found"))?;

        Ok(Response::new(row_to_song(row)))
    }

    async fn list_songs(
        &self,
        request: Request<ListSongsRequest>,
    ) -> Result<Response<ListSongsResponse>, Status> {
        let limit = sanitize_page_size(request.into_inner().page_size);

        let rows = sqlx::query_as::<_, SongRow>(
            r#"
            SELECT id, title, description, link
            FROM songs
            ORDER BY id
            LIMIT $1
            "#,
        )
        .bind(limit)
        .fetch_all(&self.pool)
        .await
        .map_err(map_db_error)?;

        let songs = rows.into_iter().map(row_to_song).collect();
        Ok(Response::new(ListSongsResponse {
            songs,
            next_page_token: String::new(),
        }))
    }

    async fn update_song(
        &self,
        request: Request<UpdateSongRequest>,
    ) -> Result<Response<Song>, Status> {
        let request = request.into_inner();
        let song = request
            .song
            .ok_or_else(|| Status::invalid_argument("update_song requires song payload"))?;
        if song.id == 0 {
            return Err(Status::invalid_argument("song id is required"));
        }

        let existing = sqlx::query_as::<_, SongRow>(
            r#"
            SELECT id, title, description, link
            FROM songs
            WHERE id = $1
            "#,
        )
        .bind(song.id as i64)
        .fetch_optional(&self.pool)
        .await
        .map_err(map_db_error)?
        .ok_or_else(|| Status::not_found("song not found"))?;

        let updated = apply_song_update_mask(&existing, &song, request.update_mask)?;
        if updated.title.trim().is_empty() {
            return Err(Status::invalid_argument("song title is required"));
        }

        let row = sqlx::query_as::<_, SongRow>(
            r#"
            UPDATE songs
            SET title = $1, description = $2, link = $3
            WHERE id = $4
            RETURNING id, title, description, link
            "#,
        )
        .bind(updated.title)
        .bind(empty_to_none(updated.description))
        .bind(empty_to_none(updated.link))
        .bind(song.id as i64)
        .fetch_one(&self.pool)
        .await
        .map_err(map_db_error)?;

        Ok(Response::new(row_to_song(row)))
    }

    async fn delete_song(
        &self,
        request: Request<DeleteSongRequest>,
    ) -> Result<Response<()>, Status> {
        let id = parse_id(&request.into_inner().name)?;

        let result = sqlx::query("DELETE FROM songs WHERE id = $1")
            .bind(id)
            .execute(&self.pool)
            .await
            .map_err(map_db_error)?;

        if result.rows_affected() == 0 {
            return Err(Status::not_found("song not found"));
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

fn empty_to_none(value: String) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(value)
    }
}

fn row_to_song(row: SongRow) -> Song {
    Song {
        id: row.id as u64,
        title: row.title,
        description: row.description.unwrap_or_default(),
        link: row.link.unwrap_or_default(),
    }
}

fn apply_song_update_mask(
    existing: &SongRow,
    incoming: &Song,
    mask: Option<prost_types::FieldMask>,
) -> Result<Song, Status> {
    let mut updated = row_to_song(existing.clone());

    let paths = mask
        .map(|mask| mask.paths)
        .unwrap_or_else(Vec::new);

    if paths.is_empty() {
        updated.title = incoming.title.clone();
        updated.description = incoming.description.clone();
        updated.link = incoming.link.clone();
        return Ok(updated);
    }

    for path in paths {
        match path.as_str() {
            "title" => updated.title = incoming.title.clone(),
            "description" => updated.description = incoming.description.clone(),
            "link" => updated.link = incoming.link.clone(),
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
    use super::{apply_song_update_mask, parse_id, row_to_song, SongRow};
    use api::pb::Song;

    #[test]
    fn parse_id_rejects_invalid_values() {
        assert!(parse_id("").is_err());
        assert!(parse_id("-1").is_err());
        assert!(parse_id("abc").is_err());
    }

    #[test]
    fn update_mask_updates_selected_fields() {
        let existing = SongRow {
            id: 1,
            title: "Old".to_string(),
            description: Some("Old desc".to_string()),
            link: Some("old".to_string()),
        };
        let incoming = Song {
            id: 1,
            title: "New".to_string(),
            description: "New desc".to_string(),
            link: "new".to_string(),
        };
        let mask = prost_types::FieldMask {
            paths: vec!["title".to_string(), "link".to_string()],
        };

        let updated = apply_song_update_mask(&existing, &incoming, Some(mask)).expect("updated");
        assert_eq!(updated.title, "New");
        assert_eq!(updated.link, "new");
        assert_eq!(updated.description, "Old desc");
    }

    #[test]
    fn row_to_song_defaults_missing_fields() {
        let row = SongRow {
            id: 2,
            title: "Title".to_string(),
            description: None,
            link: None,
        };
        let song = row_to_song(row);
        assert_eq!(song.description, "");
        assert_eq!(song.link, "");
    }
}
