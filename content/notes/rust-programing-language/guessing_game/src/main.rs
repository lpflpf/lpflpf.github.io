use std::io;

use rand::Rng;
use std::cmp::Ordering;

fn main() {
    println!("Guess the number!");

    // 默认 i32 类型
    let secret_number = rand::thread_rng().gen_range(1..=100);

    loop {
        println!("Please input youer guess.");

        let mut guess = String::new();
        io::stdin()
            .read_line(&mut guess)
            .expect("Failed to read line");

        println!("You guessed: {guess}");

        // 变量覆盖，原来的string 类型变成了 u32类型
        // match arms, 匹配 ok 和 err
        // parse 返回值 是Result；枚举类型，有 Ok, Err 两个成员
        let guess: u32 = match guess.trim().parse() {
            Ok(num) => num,     // 返回 数字
            Err(_) => continue, // 重新循环, _ 是通配符，匹配所有err
        };

        // match 和 分支(arms)
        match guess.cmp(&secret_number) {
            Ordering::Less => println!("Too small!"),
            Ordering::Greater => println!("Too big!"),
            Ordering::Equal => {
                println!("You win!");
                break; // 中断循环
            }
        }
        println!("You guessed: {guess}")
    }
}
