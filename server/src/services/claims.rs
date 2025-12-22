use serde::{Deserialize, Serialize};

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Claims {
    pub tg_id: u64,
    // unix time when the form was opened
    pub auth_date: Option<u64>,

    pub exp: usize,
    pub iat: usize,
    pub is_admin: bool,
}
