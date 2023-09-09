use std::io::Read;
use std::process::exit;

use crate::Database;

pub trait Password {
    fn get(self) -> String;
}

impl Password for String {
    fn get(self) -> String {
        self
    }
}

pub struct PasswordStdin;

impl Password for PasswordStdin {
    fn get(self) -> String {
        let mut result = String::new();
        std::io::stdin().read_to_string(&mut result).unwrap();
        result
    }
}

pub fn register<P>(db: Database, user: String, password: P)
where
    P: Password,
{
    let password = password.get();
    let encrypted_password = bcrypt::hash(password, bcrypt::DEFAULT_COST).unwrap();
    if let Err(msg) = db.add_user(user, encrypted_password) {
        println!("{msg}");
        exit(1);
    }
}

pub fn reset_password<P>(db: Database, user: String, password: P)
where
    P: Password,
{
    let password = password.get();
    let encrypted_password = bcrypt::hash(password, bcrypt::DEFAULT_COST).unwrap();
    if let Err(msg) = db.reinitialize_user(user, encrypted_password) {
        println!("{msg}");
        exit(1);
    }
}

pub fn rename(db: Database, user: String, new_name: String)
{
    if let Err(msg) = db.rename_user(user, new_name) {
        println!("{msg}");
        exit(1);
    }
}
