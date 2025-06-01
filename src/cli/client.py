import socket
import sys
import os
from dotenv import load_dotenv

def main():
    load_dotenv()
    vm_ip = os.getenv("HOST_IP_ADDR")
    vm_port = int(os.getenv("HOST_PORT"))
    
    if not vm_ip or not vm_port: 
        print("One of the Inputs Were Invalid, Exiting")
        sys.exit(1)

    try:
        client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    except socket.error as e:
        print(f"Error creating socket: {e}")
        sys.exit(1)

    print(f"Connecting to VM at {vm_ip}:{vm_port}")

    try:
        client_socket.connect((vm_ip, vm_port))
        print("Connected successfully!")
    except socket.error as e:
        print(f"Connection failed: {e}")
        sys.exit(1)

    try:
        while True:
            message = input("Enter command (e.g., 'PUT key value', 'GET key', or type 'exit' to quit): ")
            if message.lower() == 'exit':
                break
            if not message:
                continue

            client_socket.sendall((message + '\n').encode('utf-8'))

            response_parts = []
            client_socket.settimeout(2.0)
            try:
                while True:
                    data = client_socket.recv(2048)
                    if not data:
                        break
                    response_parts.append(data.decode('utf-8'))
                    if data.decode('utf-8').endswith('\n'):
                        break
            except socket.timeout:
                if not response_parts:
                    print("No response received from server (timeout).")
            client_socket.settimeout(None)

            if response_parts:
                full_response = "".join(response_parts)
                print(f"Server response: {full_response.strip()}")

    except KeyboardInterrupt:
        print("\nExiting due to user interruption.")
    except socket.error as e:
        print(f"Socket error during communication: {e}")
    finally:
        print("Closing connection.")
        client_socket.close()

if __name__ == "__main__":
    main()