.PHONY: deploy clean

deploy: bot.zip

clean:
	rm *.hlt *.log

bot.zip: MyBot.go
	zip -r bot.zip MyBot.go src
