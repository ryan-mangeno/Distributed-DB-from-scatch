#include <iostream>
#include <string>
#include <vector>
#include <cstring>
#include <sstream>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <sqlite3.h> 

constexpr const char* SOCKET_PATH = "/tmp/storage_engine.sock";
constexpr const char* DB_PATH = "storage.db";

sqlite3* db;

std::string handle_put(const std::vector<std::string>& tokens);
std::string handle_get(const std::vector<std::string>& tokens);
std::string process_command(const std::string& command);
void handle_client_connection(int client_fd);


bool initialize_database() {
    if (sqlite3_open(DB_PATH, &db)) {
        std::cerr << "C++ Storage Engine: Can't open database: " << sqlite3_errmsg(db) << std::endl;
        return false;
    } else {
        std::cout << "C++ Storage Engine: Opened database successfully" << std::endl;
    }

    // create a users  table if it dne
    const char* sql_create_table =
        "CREATE TABLE IF NOT EXISTS users ("
        "name TEXT PRIMARY KEY NOT NULL,"
        "age  INTEGER NOT NULL);";

    char* zErrMsg = 0;
    if (sqlite3_exec(db, sql_create_table, 0, 0, &zErrMsg) != SQLITE_OK) {
        std::cerr << "C++ Storage Engine: SQL error: " << zErrMsg << std::endl;
        sqlite3_free(zErrMsg);
        return false;
    }
    std::cout << "C++ Storage Engine: 'users' table is ready." << std::endl;
    return true;
}


int main() {
    if (!initialize_database()) {
        return EXIT_FAILURE;
    }

    int server_fd;
    struct sockaddr_un server_addr;

    server_fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (server_fd == -1) {
        perror("C++ Storage Engine: socket creation failed");
        return EXIT_FAILURE;
    }

    memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sun_family = AF_UNIX;
    strncpy(server_addr.sun_path, SOCKET_PATH, sizeof(server_addr.sun_path) - 1);

    unlink(SOCKET_PATH);

    if (bind(server_fd, (struct sockaddr*)&server_addr, sizeof(server_addr)) == -1) {
        perror("C++ Storage Engine: bind failed");
        close(server_fd);
        return EXIT_FAILURE;
    }

    if (listen(server_fd, 5) == -1) {
        perror("C++ Storage Engine: listen failed");
        close(server_fd);
        return EXIT_FAILURE;
    }
    std::cout << "C++ Storage Engine: Listening on UDS path: " << SOCKET_PATH << std::endl;

    while (true) {
        int client_fd = accept(server_fd, NULL, NULL);
        if (client_fd == -1) {
            perror("C++ Storage Engine: accept failed");
            continue;
        }
        handle_client_connection(client_fd);
    }

    sqlite3_close(db);
    close(server_fd);
    return EXIT_SUCCESS;
}


void handle_client_connection(int client_fd) {
    std::vector<char> buffer(1024);
    ssize_t bytes_received = recv(client_fd, buffer.data(), buffer.size() - 1, 0);

    if (bytes_received > 0) {
        buffer[bytes_received] = '\0';
        std::string command_str(buffer.data());
        std::string response_str = process_command(command_str);
        send(client_fd, response_str.c_str(), response_str.length(), 0);
    }
    close(client_fd);
}

std::string process_command(const std::string& command) {
    std::cout << "C++ Storage Engine: Received command: \"" << command << "\"" << std::endl;
    std::stringstream ss(command);
    std::string token;
    std::vector<std::string> tokens;
    while (ss >> token) {
        tokens.push_back(token);
    }

    if (tokens.empty()) {
        return "ERROR: Empty command.";
    }

    if (tokens[0] == "PUT" && tokens.size() == 4 && tokens[2] == "age") {
        return handle_put(tokens);
    } else if (tokens[0] == "GET" && tokens.size() == 2) {
        return handle_get(tokens);
    }

    return "ERROR: Unknown or malformed command. Use 'PUT <name> age <age>' or 'GET <name>'.";
}

std::string handle_put(const std::vector<std::string>& tokens) {
    std::string name = tokens[1];
    int age;
    try {
        age = std::stoi(tokens[3]);
    } catch (const std::exception& e) {
        return "ERROR: Invalid age provided.";
    }

    const char* sql_insert = "INSERT OR REPLACE INTO users (name, age) VALUES (?, ?);";
    sqlite3_stmt* stmt;

    if (sqlite3_prepare_v2(db, sql_insert, -1, &stmt, 0) != SQLITE_OK) {
        return "ERROR: Failed to prepare statement.";
    }

    sqlite3_bind_text(stmt, 1, name.c_str(), -1, SQLITE_STATIC);
    sqlite3_bind_int(stmt, 2, age);

    if (sqlite3_step(stmt) != SQLITE_DONE) {
        sqlite3_finalize(stmt);
        return "ERROR: Failed to execute statement: " + std::string(sqlite3_errmsg(db));
    }

    sqlite3_finalize(stmt);
    return "OK: User " + name + " saved.";
}

std::string handle_get(const std::vector<std::string>& tokens) {
    std::string name_to_find = tokens[1];
    const char* sql_select = "SELECT age FROM users WHERE name = ?;";
    sqlite3_stmt* stmt;

    if (sqlite3_prepare_v2(db, sql_select, -1, &stmt, 0) != SQLITE_OK) {
        return "ERROR: Failed to prepare statement.";
    }

    sqlite3_bind_text(stmt, 1, name_to_find.c_str(), -1, SQLITE_STATIC);

    if (sqlite3_step(stmt) == SQLITE_ROW) {
        int age = sqlite3_column_int(stmt, 0);
        sqlite3_finalize(stmt);
        return "OK: Found " + name_to_find + " with age " + std::to_string(age);
    } else {
        sqlite3_finalize(stmt);
        return "NOT_FOUND: User " + name_to_find + " not found.";
    }
}