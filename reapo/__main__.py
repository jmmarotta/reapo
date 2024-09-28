import sys
import json
from server import JSONRPCServer


# def main():
#     server = JSONRPCServer()
#     try:
#         while True:
#             # Read input from stdin
#             input_line = sys.stdin.readline()
#             if not input_line:
#                 break  # EOF
#             request = json.loads(input_line)
#             response = server.handle_request(request)
#             if response:
#                 sys.stdout.write(json.dumps(response) + "\n")
#                 sys.stdout.flush()
#     except KeyboardInterrupt:
#         pass
#
#
# if __name__ == "__main__":
#     main()


def main():
    server = ReapoServer()
    server.start()


if __name__ == "__main__":
    main()
