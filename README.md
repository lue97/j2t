# j2t
A simple utility to extract a structure from JSON.
# Usage
```text
➜  echo '{"name": "Sam","dislikes": ["Green eggs and ham"]}' | j2t 
.dislikes[] string "Green eggs and ham"
.name       string "Sam"
```