- tools ask for permission (for certain tools)
   - Yes
   - No, and specify what to do
```
I want certain tools to ask the user for permission before executing. For example, if the user edits a file, I want the user to be prompted to confirm that they want to edit the file before the edit is made.

Two options should be shown to the user: "Yes" and "No, and give feedback".

Come up with a plan to implement this for the tools that could write, overwite, edit, or delete files.
```

- rename reapo to clai

- make sure token counts are correct and include system prompt I think it's divided 10 and base is 1M if necessary or 200k

- no scrolling in chat view
  - mouse scroll for ui messages by default Why can I not scroll in the chat views UIMessages? is there a way for me to persist messages in the terminal so that I don't lose them?

- add angle |_ to indicate which message the diff belongs to
  - Help me come up with a plan to better abstract the messages I want

- there is too much happening in update.go

- plan and build modes
  - planning tool?

- write tests for main parts of agent
  - agent
  - tools
  - session

- style markdown in chat view

- better error handling for tools

- ensure file is read before write (ensure)
- limit file read size to 2000 lines (ensure)

- adjust prompts according to the lastest claude code version
  - https://cchistory.mariozechner.at/
  - https://mariozechner.at/posts/2025-08-03-cchistory/?utm_source=tldrai

- bash should have a timeout

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

- OpenAPI provider

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

- which-key style help
