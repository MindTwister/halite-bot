deploy: bot.zip


bot.zip: MyBot.go
	zip -r bot.zip MyBot.go src
