**Time Zone**

Commands:

 `/tz help`

print this help text

---


 `/tz now <location>`

prints timezone info for `location`

locations included are:
    
- "usa"

Example:

```text
/tz now usa
```
```text
01:50 EDT America/New_York
00:50 CDT America/Chicago
23:50 MDT America/Denver
22:50 MST America/Phoenix
22:50 PDT America/Los_Angeles
```





---

 `/tz HH:MM <tz from> <tz to>`

display `HH:MM` time as `<tz to>` timezone.

Example:
```text
/tz 18:30 est pst
```


```text
Your Time: 18:30 EST 6:30pm EST
Their Time: 15:30 EST 3:30pm EST
```

---