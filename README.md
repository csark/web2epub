# web-epub
Take a webpage and turn its content into an ereader ready epub



### Testing 
Current testing strings

General Conference
go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/10?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/5uahv05h1s6416y49vw745z70juiiffhiq0vn8a2/full/%21250%2C/0/default"

Scriptures
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/bofm?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/59fa03a8250ea7aea58e9f3515031ea47b6ab7eb/full/%21250%2C/0/default"
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/dc-testament?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/d9930401562cdd688233134c5f20a5d75b968b14/full/%21250%2C/0/default"
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/nt?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/7d175abca40ddfa795593e4f713a44489acc6cd5/full/%21250%2C/0/default"

Come, Follow Me
go run main.go -url "https://www.churchofjesuschrist.org/study/manual/come-follow-me-for-home-and-church-doctrine-and-covenants-2025?lang=eng" -module cfm -cover "https://www.churchofjesuschrist.org/imgs/c63fc6d8f3fc11ed9b72eeeeac1e0c3d06b3957c/full/%21250%2C/0/default"


go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2024/10?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/s4c6koq1axflec6ni6s2kdoe50xv7lnhy3p1ckry/full/%21250%2C/0/default"


go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/speakers/dallin-h-oaks?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/db96515429b848527e30220394ba6299c297236a/full/!250%2C300/0/default"