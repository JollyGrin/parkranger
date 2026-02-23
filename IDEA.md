I am increasingly using git worktrees to help parallelize my work. I'm also using tmux to make it easy to have different work sessions with split panes for code and claude.

Yet I have yet to find a good tool for organizing/visualizing all my active sessions.

I'm right now using:

- tmux new -s <current-task-name>
- ccmanager > create new worktree
- workmux open <worktreename>

then i ctrl-b to release and hit w to view all my sessions. This is a real bummer.

I like the view of ccmanager but it only works for 1 repo, and if i close it, it loses the status of all the claude sessions it had open.

My ideal workflow would be:

- target a repo and worktree (or create one - similar to ccmanager)
- opens a tmux (new/loaded) with a split pane for vi/claude {perhaps with a cc- prefix to isolate these tmux groups}. The claude & code should both be "in" the worktree folder to prevent confusion
- overview mode to see claude agent status of these different tmux sessions. Should be able to pickup when a claude session is active (even if closed and reopened). If many claude windows in a session, picks up the most idle state (to alert when input is needed)
- bonus: if we can preview the actual claude session, such as seeing the text actively streaming from a response
- be able to add repos/worktrees and remove them
- later: merge functionalities

My ideal techstack is to use go. Find the best tui tools

And perhaps before we even begin the tui, we need a solid engine which can:

- query tmux sessions for claude sessions
- understand when a claude agent is busy, waiting, idle
- active polling without degrading performance. Absolutely must be minimal/not have memory leaks.
- when querying claude sessions, should save the conversation has for quick pickup later in case a session closes, its visible on that worktree again.

Where do we begin for this type of setup?
