# web-epub
Take a webpage and turn its content into an ereader ready epub



### Testing 
Current testing strings

General Conference
go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/04?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/1hsgsztu81qrzo6i6u7l0g7d8zvop7dg93ctfug4/full/%21320%2C/0/default"

Scriptures
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/bofm?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/59fa03a8250ea7aea58e9f3515031ea47b6ab7eb/full/%21250%2C/0/default"
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/dc-testament?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/d9930401562cdd688233134c5f20a5d75b968b14/full/%21250%2C/0/default"
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/nt?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/7d175abca40ddfa795593e4f713a44489acc6cd5/full/%21250%2C/0/default"

Come, Follow Me
go run main.go -url "https://www.churchofjesuschrist.org/study/manual/come-follow-me-for-home-and-church-doctrine-and-covenants-2025?lang=eng" -module cfm -cover "https://www.churchofjesuschrist.org/imgs/c63fc6d8f3fc11ed9b72eeeeac1e0c3d06b3957c/full/%21250%2C/0/default"