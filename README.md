# web-epub
Take a webpage and turn its content into an ereader ready epub



### Testing 
Current testing strings

General Conference
go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/04?lang=eng" -cover "https://www.churchofjesuschrist.org/imgs/1hsgsztu81qrzo6i6u7l0g7d8zvop7dg93ctfug4/full/%21320%2C/0/default"

Scriptures
go run main.go -url "https://www.churchofjesuschrist.org/study/scriptures/bofm?lang=eng" -module scriptures -cover "https://www.churchofjesuschrist.org/imgs/59fa03a8250ea7aea58e9f3515031ea47b6ab7eb/full/%21250%2C/0/default"