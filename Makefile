EXECUTABLE := terraform-provider-axonops

build:
	go build -o $(EXECUTABLE)

buildnrun:
	go build -o $(EXECUTABLE)
	rm terraform.tfstate
	terraform apply -auto-approve
