# web-epub
Take a webpage and turn its content into an ereader ready epub

Current testing strings


go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/04/13holland?lang=eng" -depth 1 -output conference.epub
go run main.go -url "https://www.churchofjesuschrist.org/study/general-conference/2025/04?lang=eng" -depth 2 -output conference.epub