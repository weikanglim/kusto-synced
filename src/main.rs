use clap::{Parser, Subcommand};

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
}

fn main() {
    let cli = Cli::parse();

    match &cli.command {
        Some(Commands::Sync { cluster }) => {
            // do some parsing here
            sync(&cluster);
        },

        None => todo!(),
    }
}

fn sync(_cluster: &str) {

}