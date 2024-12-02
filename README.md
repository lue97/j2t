# j2t
A simple utility to extract a structure from JSON.
# Usage
```text
usage: j2t [-h|--help] [-o|--output "<value>"] [-i|--input "<value>"]
           [-f|--format (list|json|csv)] [-P|--prefix "<value>"] [-H|--headers]
           [-m|--merge] [-n|--numeric]

           Lists the fields in a JSON string

Arguments:

  -h  --help     Print help information
  -o  --output   Sets the output file. Reads from STDIN by default
  -i  --input    Sets the input file. Writes to STDOUT by default
  -f  --format   Output format. Default: list
  -P  --prefix   Field prefix
  -H  --headers  If headers should be printed. Only applicable for `list` and
                 `csv` format
  -m  --merge    Merges type and content for fields with multiple types. Only
                 applicable for `list` and `csv` format
  -n  --numeric  Categorize `number` into `number_int` and `number_float`
```
Simple usage
```text
➜  echo '{"name": "Sam", "dislikes": ["Green egg", "Ham"]}' | j2t
.dislikes[] string "Green egg"
.name       string "Sam"
```
Print CSV with headers
```text
➜  echo '{"name": "Sam", "dislikes": ["Green egg", "Ham"]}' | j2t -f csv -H
field,type,content
.dislikes[],string,"""Green egg"""
.name,string,"""Sam"""
```
Supports multi-type arrays
```text
➜  echo '[1, "two"]' | j2t -n
[]    string     "two"
[]    number_int 1
```
