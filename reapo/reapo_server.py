import sys
import json


class ReapoServer:
    def __init__(self):
        self.handlers = {
            "initialize": self.initialize,
            "initialized": self.initialized,
            "shutdown": self.shutdown,
            "exit": self.exit,
            "indexGitRepo": self.index_git_repo,
            "addFileDocumentationToRepo": self.add_file_documentation_to_repo,
            "addURLDocumentationToRepo": self.add_url_documentation_to_repo,
            "addSiteDocumentationToRepo": self.add_site_documentation_to_repo,
            "retrieveRepoContext": self.retrieve_repo_context,
            "retrieveDocumentationContext": self.retrieve_documentation_context,
            "retrieveContext": self.retrieve_context,
        }
        self.running = True

    # Starts an IO server that listens for requests from the client
    def start(self):
        while self.running:
            request_line = sys.stdin.readline()
            if not request_line:
                break  # EOF
            try:
                request = json.loads(request_line)
                response = self.handle_request(request)
                if response is not None:
                    sys.stdout.write(json.dumps(response) + "\n")
                    sys.stdout.flush()
            except Exception as e:
                error_response = {
                    "jsonrpc": "2.0",
                    "error": {
                        "code": -32603,
                        "message": str(e),
                    },
                    "id": request.get("id", None),
                }
                sys.stdout.write(json.dumps(error_response) + "\n")
                sys.stdout.flush()

    def handle_request(self, request):
        method = request.get("method")
        handler = self.handlers.get(method)
        if handler:
            return handler(request)
        else:
            return {
                "jsonrpc": "2.0",
                "error": {
                    "code": -32601,
                    "message": "Method not found",
                },
                "id": request.get("id", None),
            }

    # Handlers
    def initialize(self, request):
        pass

    def initialized(self, request):
        pass

    def shutdown(self, request):
        pass

    def exit(self, request):
        pass

    def index_git_repo(self, request):
        pass

    def add_file_documentation_to_repo(self, request):
        pass

    def add_url_documentation_to_repo(self, request):
        pass

    def add_site_documentation_to_repo(self, request):
        pass

    def retrieve_repo_context(self, request):
        pass

    def retrieve_documentation_context(self, request):
        pass

    def retrieve_context(self, request):
        pass
