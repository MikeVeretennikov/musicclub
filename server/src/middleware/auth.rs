use std::sync::Arc;

use jsonwebtoken::{Algorithm, Validation, decode};
use tonic::body::Body;
use tonic::codegen::http::Request as HttpRequest;
use tonic::{Result, Status};
use tonic_middleware::RequestInterceptor;

use crate::services::{Claims, Keys};

#[derive(Clone, Debug)]
pub struct AuthInterceptor {
    keys: Arc<Keys>,
}

impl AuthInterceptor {
    pub fn new(secret_key: &[u8]) -> Self {
        Self {
            keys: Arc::new(Keys::new(secret_key)),
        }
    }

    fn decode(&self, token: &str) -> Result<Claims, Status> {
        let mut validation = Validation::new(Algorithm::HS256);
        validation.validate_exp = true;
        let data = decode::<Claims>(token, &self.keys.decoding, &validation)
            .map_err(|_| Status::unauthenticated("invalid token"))?;
        Ok(data.claims)
    }
}

#[tonic::async_trait]
impl RequestInterceptor for AuthInterceptor {
    async fn intercept(&self, req: HttpRequest<Body>) -> Result<HttpRequest<Body>, Status> {
        if !req.uri().path().ends_with("/CreateConcert") {
            return Ok(req);
        }

        let auth_header = req
            .headers()
            .get("authorization")
            .and_then(|value| value.to_str().ok())
            .ok_or_else(|| Status::unauthenticated("authorization header required"))?;
        let token = auth_header.strip_prefix("Bearer ").unwrap_or(auth_header);
        let claims = self.decode(token)?;

        let mut req = req;
        let header_value = tonic::codegen::http::HeaderValue::from_str(&claims.tg_id.to_string())
            .map_err(|_| Status::internal("invalid user id header"))?;
        req.headers_mut().insert("x-user-id", header_value);
        Ok(req)
    }
}
