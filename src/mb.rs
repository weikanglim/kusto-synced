
// consumeSeparator advances the std::io::BufRead by a line,
// returning characters after separation found on the current line.
// If no separation is found on the current line,
// the std::io::BufRead is still advanced, and an empty string is returned.
fn consumeSeparator(mut reader: impl std::io::BufRead) -> Result<String, Box<dyn std::error::Error>> {
    let mut current = String::new();
    let res = reader.read_line(&mut current)?;
    if res == 0 {
        return Ok("".to_string())
    }

    for (i, char) in current.char_indices() {
        if !char.is_ascii_whitespace() {
            return Ok(current.get(i..current.len()).unwrap().to_string())
        }
    }

    // EOL, 
    Ok("".to_string())
}