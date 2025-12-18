mod grpc;

use api::pb::{
    auth_service_server, concert_service_server, participation_service_server, song_service_server,
};
use env_logger::Env;
use sqlx::postgres::PgPoolOptions;
use tonic::{Result, transport::Server};
use tonic_middleware::{MiddlewareLayer, RequestInterceptorLayer};

use crate::grpc::{
    auth::{AuthInterceptor, AuthServer},
    concert::ConcertServer,
    middleware::{AdminOnlyMiddleware, LoggingMiddleware},
    participation::ParticipationServer,
    song::SongServer,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    env_logger::Builder::from_env(Env::default().default_filter_or("debug")).init();
    dotenvy::dotenv().ok();
    let addr = "[::1]:6969".parse()?;
    let database_url = database_url_from_env()?;
    let pool = PgPoolOptions::new()
        .max_connections(8)
        .connect(&database_url)
        .await?;
    let admin_ids = load_admin_ids()?;
    let jwt_secret = std::env::var("JWT_SECRET")
        .or_else(|_| std::env::var("BOT_TOKEN"))
        .map_err(|_| "JWT_SECRET or BOT_TOKEN must be set")?;
    let jwt_ttl_seconds: u64 = std::env::var("JWT_TTL_SECONDS")
        .ok()
        .and_then(|value| value.parse().ok())
        .unwrap_or(60 * 60);
    let auth_interceptor = AuthInterceptor::new(jwt_secret.as_bytes());
    let admin_middleware = AdminOnlyMiddleware::new(admin_ids.clone());

    log::info!("Server is running at {addr}");
    Server::builder()
        .layer(MiddlewareLayer::new(LoggingMiddleware::default()))
        .layer(RequestInterceptorLayer::new(auth_interceptor))
        .layer(MiddlewareLayer::new(admin_middleware))
        .add_service(auth_service_server::AuthServiceServer::new(
            AuthServer::new(
                jwt_secret.as_bytes(),
                admin_ids,
                std::time::Duration::from_secs(jwt_ttl_seconds),
            ),
        ))
        .add_service(song_service_server::SongServiceServer::new(
            SongServer::new(pool.clone()),
        ))
        .add_service(concert_service_server::ConcertServiceServer::new(
            ConcertServer::new(pool.clone()),
        ))
        .add_service(
            participation_service_server::ParticipationServiceServer::new(
                ParticipationServer::new(pool.clone()),
            ),
        )
        .serve(addr)
        .await?;

    Ok(())
}

fn database_url_from_env() -> Result<String, Box<dyn std::error::Error>> {
    if let Ok(url) = std::env::var("DATABASE_URL") {
        return Ok(url);
    }

    if let Ok(url) = std::env::var("POSTGRES_URL") {
        return Ok(url
            .replace("postgresql+asyncpg://", "postgres://")
            .replace("postgresql://", "postgres://"));
    }

    let user = std::env::var("POSTGRES_USER")?;
    let password = std::env::var("POSTGRES_PASSWORD")?;
    let host = std::env::var("POSTGRES_HOST")?;
    let db = std::env::var("POSTGRES_DB")?;
    let port = std::env::var("POSTGRES_PORT").unwrap_or_else(|_| "5432".to_string());

    Ok(format!("postgres://{user}:{password}@{host}:{port}/{db}"))
}

fn load_admin_ids() -> Result<std::collections::HashSet<u64>, Box<dyn std::error::Error>> {
    let raw = std::env::var("ADMIN_IDS")?;
    let ids: Vec<u64> = serde_json::from_str(&raw)?;
    Ok(ids.into_iter().collect())
}

#[cfg(test)]
mod tests {
    use super::database_url_from_env;

    #[test]
    fn builds_database_url_from_parts() {
        unsafe {
            std::env::remove_var("DATABASE_URL");
            std::env::remove_var("POSTGRES_URL");
            std::env::set_var("POSTGRES_USER", "user");
            std::env::set_var("POSTGRES_PASSWORD", "pass");
            std::env::set_var("POSTGRES_HOST", "localhost");
            std::env::set_var("POSTGRES_DB", "db");
            std::env::set_var("POSTGRES_PORT", "5433");
        }

        let url = database_url_from_env().expect("url");
        assert_eq!(url, "postgres://user:pass@localhost:5433/db");
    }

    #[test]
    fn respects_postgres_url_override() {
        unsafe {
            std::env::remove_var("DATABASE_URL");
            std::env::set_var(
                "POSTGRES_URL",
                "postgresql+asyncpg://user:pass@localhost:5432/db",
            );
        }

        let url = database_url_from_env().expect("url");
        assert_eq!(url, "postgres://user:pass@localhost:5432/db");
    }
}
