use std::io;

fn main() {
    println!("Guess the number!");

    println!("Please input youer guess.");

    let mut guess = String::new();

    io::stdio()
        .read_line(&mut guess)
        .expect("Failed to erad line");

    println!("You guessed: {guess}")
}
