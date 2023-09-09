use postgres::{Client, NoTls};
use std::env::var as getenv;

pub struct Database {
    connection: Client,
}

impl Database {
    pub fn new(
        address: Option<String>,
        port: Option<u16>,
        user: Option<String>,
        password: Option<String>,
        db_name: Option<String>,
    ) -> Self {
        let address = address.unwrap_or_else(|| {
            getenv("premises_controlPanel_postgres_address").unwrap_or("localhost".to_string())
        });
        let port = port.unwrap_or_else(|| {
            getenv("premises_controlPanel_postgres_port")
                .ok()
                .map(|v| v.parse().unwrap())
                .unwrap_or(5432)
        });
        let user = user.unwrap_or_else(|| {
            getenv("premises_controlPanel_postgres_user").unwrap_or("premises".to_string())
        });
        let password =
            password.unwrap_or_else(|| getenv("premises_controlPanel_postgres_password").unwrap());
        let db_name = db_name.unwrap_or_else(|| {
            getenv("premises_controlPanel_postgres_dbName").unwrap_or("premises".to_string())
        });

        let connection = Client::connect(
            &format!("host={address} port={port} user={user} password={password} dbname={db_name}"),
            NoTls,
        )
        .unwrap();

        Self { connection }
    }
}

impl Database {
    pub fn add_user(mut self, username: String, encrypted_password: String) -> Result<(), String> {
        match self.connection.execute(
            "INSERT INTO users (name, password, initialized) VALUES ($1, $2, 't')",
            &[&username, &encrypted_password],
        ) {
            Ok(_) => Ok(()),
            Err(err) => Err(err.as_db_error().unwrap().message().to_string()),
        }
    }
}
