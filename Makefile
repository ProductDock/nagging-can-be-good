.PHONY: dep bootstrap synth deploy destroy

export SSH_PUB_KEY := $(shell cat ./live.pub)

dep:
	npm install

bootstrap:
	cdk bootstrap

synth:
	cdk synth

deploy:
	cdk deploy

destroy:
	cdk destroy