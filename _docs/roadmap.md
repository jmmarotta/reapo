- tools have output
  - follow up: remove ...

- Use Meyer's diff algorithm and style diff with lipgloss

- chat is too large and collides with border of input
- no scrolling in chat view

- tools ask for permission (for certain tools)
   - Yes
   - No, and specify what to do

- style markdown in chat view

- plan and build modes
  - planning tool

- bash should have a timeout

- adjust prompts according to the lastest claude code version
  - https://cchistory.mariozechner.at/
  - https://mariozechner.at/posts/2025-08-03-cchistory/?utm_source=tldrai

- certain tools always need permission
  - set in config
  - probably for bash tool

- ensure:
  - mark todo as in progress
  - cannot write to file before read

- optimize bubbletea
  - https://leg100.github.io/en/posts/building-bubbletea-programs/

- Anthropic prompt caching

- put text input in middle until there is more text

- refactor update.go
  - refactor out slash commands "/" and file/dir references "@"

- cancel API calls

- recover from errors in request (retry)

- Cerebras / OpenAPI provider

- Improve in progress request (padding and color)
  - time taken
  - tokens ingested

- persist sessions

- add conversation history to tui model to keep track of conversations
  - building for now. Probably need a session model

- allow different models in different modes

- Drill down to individual files with chat on left and file on right

- everything above a certain line should be static

- summary prompt should be with agent

- Fix startup time

- Add :q and :quit

- Ctrl-c does not exit on first attempt

- markdown rendering

- write tests

- summary of what’s happening

- successful tool invocation updates tool bullet color

- add more logging

- task should have prompt in .txt

- vim-improvements

- refactor tui/update.go

- refactor agent and include update stuff
