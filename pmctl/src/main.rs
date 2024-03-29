mod database;
mod user;

use clap::{Parser, Subcommand};
use database::Database;
use std::process::exit;

#[derive(Subcommand)]
enum UserCommand {
    #[clap(name = "add", about = "Add a new user")]
    Add {
        #[clap(short = 'u', long = "username", required = true, help = "Username")]
        user: String,
        #[clap(short = 'p', long = "password", help = "Password")]
        password: Option<String>,
        #[clap(long = "password-stdin", help = "Read password from stdin")]
        password_stdin: bool,
    },
    #[clap(name = "reset-password", about = "Reset password for an existing user")]
    ResetPassword {
        #[clap(short = 'u', long = "username", required = true, help = "Username")]
        user: String,
        #[clap(short = 'p', long = "password", help = "New password")]
        password: Option<String>,
        #[clap(long = "password-stdin", help = "Read password from stdin")]
        password_stdin: bool,
    },
    #[clap(name = "rename", about = "Rename an existing user")]
    Rename {
        #[clap(short = 'u', long = "username", required = true, help = "Username")]
        user: String,
        #[clap(short = 't', long = "new-name", required = true, help = "New username")]
        new_name: String,
    },
}

#[derive(Subcommand)]
enum RootCommand {
    #[clap(subcommand)]
    User(UserCommand),
}

#[derive(Parser)]
struct Options {
    #[clap(subcommand)]
    command: RootCommand,
    #[clap(long = "database-adddress", help = "Database address")]
    db_address: Option<String>,
    #[clap(long = "database-port", help = "Database port")]
    db_port: Option<u16>,
    #[clap(long = "database-user", help = "Database user")]
    db_user: Option<String>,
    #[clap(long = "database-password", help = "Database password")]
    db_password: Option<String>,
    #[clap(long = "database-name", help = "Database name")]
    db_name: Option<String>,
}

fn main() {
    let options = Options::parse();

    let database = Database::new(
        options.db_address,
        options.db_port,
        options.db_user,
        options.db_password,
        options.db_name,
    );

    match options.command {
        RootCommand::User(user_command) => match user_command {
            UserCommand::Add {
                user,
                password,
                password_stdin,
            } => match (password, password_stdin) {
                (_, true) => user::register(database, user, user::PasswordStdin),
                (Some(password), false) => user::register(database, user, password),
                (None, false) => {
                    eprintln!("Neither --password=... nor --password-stdin is provided");
                    exit(1);
                }
            },
            UserCommand::ResetPassword {
                user,
                password,
                password_stdin,
            } => match (password, password_stdin) {
                (_, true) => user::reset_password(database, user, user::PasswordStdin),
                (Some(password), false) => user::reset_password(database, user, password),
                (None, false) => {
                    eprintln!("Neither --password=... nor --password-stdin is provided");
                    exit(1);
                }
            },
            UserCommand::Rename { user, new_name } => user::rename(database, user, new_name),
        },
    }
}
