use serde::{Deserialize, Serialize};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum GclawError {
    #[error("Configuration error: {0}")]
    Config(String),
    #[error("Provider error: {0}")]
    Provider(String),
    #[error("Channel error: {0}")]
    Channel(String),
    #[error("Database error: {0}")]
    Database(#[from] rusqlite::Error),
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),
    #[error("Tool execution error: {0}")]
    ToolExecution(String),
}

pub type Result<T> = std::result::Result<T, GclawError>;
