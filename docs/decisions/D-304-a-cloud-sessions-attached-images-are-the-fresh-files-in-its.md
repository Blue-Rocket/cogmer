# D-304 — A cloud session's attached images are the fresh files in its uploads folder that it has not sent

**Date:** 2026-11-26 · **Status:** active · **Areas:** capture, hosts, sync

**Decision.** A daemon that is a cloud session's sends with a prompt each image saved in the container's uploads folder, `/mnt/user-data/uploads`, in the 30 seconds around the prompt, whose bytes this session has not sent with an earlier prompt (D-213, a pasted image is sent to peers and each receiver stores it as a file). A Cowork cloud task puts no placeholder in the text and no number in the file's name, so the time the file was saved and its hash say which images belong to the prompt. Each is numbered after the images the session has sent. The daemon reads no file of another kind, and no image saved earlier.

**Rejected.**
- *Sending every image in the folder.* The folder holds everything the user attached to the task, and a colleague would be given pictures the user never sent to the room.
- *Taking the newest image in a folder that sessions share.* D-214 (an image a host shows after the prompt is sent in a second event) rejects it, since nothing ties such a file to the session. A task's uploads folder is its own, as the Limits say, so a file there is tied to the session.
- *Waiting for a placeholder in the prompt.* The task's prompt carries none, so no image would ever be sent.
- *Reading the session's transcript for the image blocks.* A Cowork cloud task's hooks send no transcript path (D-295, a turn with no transcript path is recorded from the Stop hook's last message).

**Limits.** An image attached more than 30 seconds before its prompt reaches the daemon is not sent. A task keeps the time a file was saved through a sleep, so an image saved before the session joined its room is not offered afterwards. The folder is taken to be the task's alone, since a task runs in a container of its own, and a container that held two sessions would send each the other's attachments. No behavior check covers the folder, the missing placeholder or the file time, since no probe runs in a cloud task (D-297, the behavior checks are not run in a cloud container). On 2026-11-26 a Cowork cloud task on build 0.11.8 sent a picture attached to a prompt, and the room page of the same account drew it. No colleague on another machine has seen one. This decision relies on D-213, whose status is not built.

**Revisit when** a cloud host names the images a prompt carries, in the prompt or in a field of the hook.
