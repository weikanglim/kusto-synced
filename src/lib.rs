use std::{path::PathBuf, fs, io::{self, BufRead, Read, Write}};

use anyhow::{Context};
use lazy_static::lazy_static;
use regex::Regex;

#[derive(Debug)]
struct ParseError {
    line: u32,
    col: u32,
    path: PathBuf,
    detail: String
}

impl std::error::Error for ParseError {
}

impl std::fmt::Display for ParseError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} ({},{}): {}", self.path.display(), self.line, self.col, self.detail)
    }
}

pub fn buildFile(
    entry: walkdir::DirEntry,
    root: &PathBuf) -> Result<(), Box<dyn std::error::Error>> {
    let name = entry.path().file_name().unwrap();
    let relative_dir = entry.path().parent().unwrap().strip_prefix(root).unwrap();
    let relative_name = relative_dir.join(name);
    let relative_out_name = format!(".out/{}", relative_name.display());
    let out_dir = root.join(".out").join(relative_dir);
    fs::create_dir_all(&out_dir).expect("failed creating .out directory");

    let file = fs::File::open(entry.path()).unwrap();
    let mut bufio = io::BufReader::new(file);

    let mut out_file = fs::File::create(out_dir.join(name)).expect("error creating output file");
    let mut comment_buf = Vec::new();
    let mut line = 0;
    let mut curr = String::new();
    loop {
        let mut innercurr = String::new();
        let bytes = bufio.read_line(&mut innercurr).expect("error reading file");
        if bytes == 0 {
            return Err(Box::new(ParseError{
                line,
                col: 0,
                path: relative_name,
                detail: "no 'let' statement found after reaching EOF".to_string(),
            }));
        }

        if innercurr.trim_start().starts_with("let") {
            curr = innercurr;
            break
        }

        comment_buf.push(innercurr);
        line += 1;
    }

    let mut index = comment_buf.len()-1;
    let mut doc = String::new();
    loop {
        let a  = &comment_buf[index];
        let b = a.trim();
        if b.is_empty() {
            break
        }

        if let Some(c) = b.strip_prefix("//") {
            doc.push_str(c.trim_start());
            doc.push(' ');
        }

        if index == 0 {
            break
        }
        index -= 1;
    }

    doc = doc.trim_end().to_string();
    lazy_static! {
        static ref RE: Regex = Regex::new(r#"let\s+([\w\d_]+)\s+=\s+(.+)"#).unwrap();
    }

    let mut remain = curr;
    let res = bufio.read_to_string(&mut remain).unwrap();
    let tomatch = remain.clone();
    let captures = match RE.captures(&tomatch) {
        Some(c) => c,
        None => return Err(Box::new(ParseError{
            line,
            col: 0,
            path: relative_name,
            detail: format!("could not parse let statement based on regex: {}", RE.as_str()).to_string(),
        }))
    };

    let name = captures.get(1).with_context(|| format!("{}: missing name", relative_name.display()))?;
    let args = captures.get(2).with_context(|| format!("{}: missing args", relative_name.display()))?;

    // `.create-or-alter function with (folder = "${relativeFolder}", docstring = "${docstring}") ${name}${args}`     
    out_file.write(
        format!(
            ".create-or-alter function with (folder=\"{}\", docstring=\"{}\") {}{}",
            relative_dir.display(), doc, name.as_str(), args.as_str()).as_bytes())
            .with_context(|| format!("writing to {}", relative_out_name))?;

    let end = captures.get(0).unwrap().end();
    let remainder = remain.get(end..remain.len()).unwrap();
    out_file.write_all(remainder.as_bytes()).with_context(|| format!("writing to {}", relative_out_name))?;
    out_file.flush().unwrap();

    Ok(())
}
