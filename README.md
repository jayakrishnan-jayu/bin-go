# bin-go
bin-go is a command-line game implemented using Golang. It is a simple game that generates a 5x5 grid of random numbers and players take turns crossing off numbers until one player has crossed off 5 rows or column combined.

## Getting Started
1. Clone the repository
2. Navigate to the root directory of the project.
3. Start the game server by running `go run cmd/server/server.go`.
4. Players can connect to the server by running `go run cmd/client/client.go -i [server_ip] -u "[Username]"`. Replace `[server_ip]` with the IP address of the machine running the server.
5. Once all the players have connected, the game can be started by typing `s` and pressing enter in the terminal where the server process is running.

## How To Play
1. Each player will be assigned a 5x5 grid of random numbers ranging from 1 to 25.
2. Players take turns providing a number from their grid that they wish to cross off, the same number will be crosesed from other players board.
3. The first player to cross off 5 rows or column combined wins the game.

## Screenshot
<img width="1191" alt="Screenshot 2023-02-16 at 3 30 25 PM" src="https://user-images.githubusercontent.com/25554170/219333472-774e03f8-8857-4e3b-8612-7bb1192d5a1c.png">
<img width="1177" alt="Screenshot 2023-02-16 at 3 31 02 PM" src="https://user-images.githubusercontent.com/25554170/219333498-cbe57898-4792-435a-bf5e-b5027343dad0.png">


## Contributing
As this project was created for learning purposes, it may have room for improvement. If you would like to contribute to this project, feel free to fork the repository and make changes. If you find any issues with the code, you can open an issue on the repository.


## License
This project is released under the MIT License. Feel free to use and modify the code as per the terms of this license.(see [LICENSE](LICENSE)).
