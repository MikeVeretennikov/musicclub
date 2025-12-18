use std::collections::HashSet;
use std::time::Instant;

use tonic::body::Body;
use tonic::codegen::http::Request;
use tonic::codegen::http::Response;
use tonic_middleware::Middleware;
use tonic_middleware::ServiceBound;

#[derive(Debug, Default, Clone)]
pub struct LoggingMiddleware;

#[tonic::async_trait]
impl<S> Middleware<S> for LoggingMiddleware
where
    S: ServiceBound,
    S::Future: Send,
{
    async fn call(&self, req: Request<Body>, mut service: S) -> Result<Response<Body>, S::Error> {
        let start_time = Instant::now();
        let remote_addr = req.uri().path().to_string().clone();

        let result = service.call(req).await?;

        let elapsed_time = start_time.elapsed();

        log::info!("{} completed in {:?}", remote_addr, elapsed_time);

        Ok(result)
    }
}

#[derive(Clone, Debug)]
pub struct AdminOnlyMiddleware {
    admin_ids: HashSet<u64>,
}

impl AdminOnlyMiddleware {
    pub fn new(admin_ids: HashSet<u64>) -> Self {
        Self { admin_ids }
    }
}

#[tonic::async_trait]
impl<S> Middleware<S> for AdminOnlyMiddleware
where
    S: ServiceBound,
    S::Future: Send,
{
    async fn call(&self, req: Request<Body>, mut service: S) -> Result<Response<Body>, S::Error> {
        if req.uri().path().ends_with("/CreateConcert") {
            println!("{:?}", req.headers());
            let user_id = req
                .headers()
                .get("x-user-id")
                .and_then(|value| value.to_str().ok())
                .and_then(|value| value.parse::<u64>().ok());
            println!("{:?}", user_id);

            if user_id.is_none() || !self.admin_ids.contains(&user_id.expect("checked")) {
                let response = tonic::Status::permission_denied("admin required").into_http();
                return Ok(response);
            }
        }

        service.call(req).await
    }
}
