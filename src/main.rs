use std::{os, fs::{FileType, self, File}, env::current_dir, path::PathBuf, io::{self, BufRead}};

use anyhow::{Context, Result};
use lazy_static::lazy_static;
use clap::{Parser, Subcommand};
use regex::Regex;
use walkdir::WalkDir;

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    /// Use verbose output (-vv for very verbose)
    #[arg(short, long, action = clap::ArgAction::Count)]
    verbosity: u8,

    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Synchronizes all stored functions and tables to the target Kusto cluster.
    Sync {
        /// The cluster to sync to.
        #[arg(short, long)]
        cluster: String
    },

    ///
    Build {

    }
}

#[derive(Debug)]
struct CustomError(String);

impl std::error::Error for CustomError {
    fn description(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for CustomError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let cli = Cli::parse();

    match &cli.command {
        Some(Commands::Sync { cluster }) => {
            // do some parsing here
            sync(&cluster);
        },

        Some(Commands::Build {}) => {
            // do some building here
            let cwd = current_dir().unwrap();
            build(&cwd)?;
        },
        None => todo!(),
    }

    Ok(())
}

fn build(root: &PathBuf) -> Result<(), Box<dyn std::error::Error>> {
    for entry in WalkDir::new(root).into_iter().filter_map(|e| e.ok()) {
        if entry.path().display().to_string().contains(".out") {
            continue
        }
        if entry.file_type().is_file() && entry.path().extension().unwrap_or_default() == "csl" {
            kusto_sync_tool::buildFile(entry, root)?;
        }
    }

    Ok(())
}

fn sync(_cluster: &str) {

}