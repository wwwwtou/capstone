# TikTok Glocal Infrastructure as Code (AWS Mockup)
# This demonstrates the planned cloud deployment architecture for the defense.

provider "aws" {
  region = "ap-southeast-1"
}

# 1. Network: VPC with Public/Private Subnets
resource "aws_vpc" "recsys_vpc" {
  cidr_block           = "10.0.0.0-16"
  enable_dns_hostnames = true

  tags = {
    Name = "tiktok-recsys-vpc"
  }
}

# 2. Compute: EC2 for Recommendation Engine (Application Server)
resource "aws_instance" "app_server" {
  ami           = "ami-0c2af51e273e592c0" # Amazon Linux 2
  instance_type = "t3.medium"
  subnet_id     = aws_subnet.private_subnet.id

  vpc_security_group_ids = [aws_security_group.app_sg.id]

  tags = {
    Name = "RecSys-App-Server"
  }
}

# 3. Database: RDS PostgreSQL Instance
resource "aws_db_instance" "postgres" {
  allocated_storage    = 20
  engine               = "postgres"
  instance_class       = "db.t3.micro"
  db_name              = "recsys"
  username             = "admin"
  password             = var.db_password
  parameter_group_name = "default.postgres15"
  
  # Security: Isolating DB in Private Subnet
  db_subnet_group_name = aws_db_subnet_group.default.name
  vpc_security_group_ids = [aws_security_group.db_sg.id]
  
  skip_final_snapshot  = true
}

# 4. Security Groups: Implementation of Least Privilege
resource "aws_security_group" "app_sg" {
  name        = "app-server-sg"
  vpc_id      = aws_vpc.recsys_vpc.id

  # Inbound from Load Balancer only
  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["10.0.1.0-24"] # Public Subnet Range
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0-0"]
  }
}

resource "aws_security_group" "db_sg" {
  name   = "rds-sg"
  vpc_id = aws_vpc.recsys_vpc.id

  # Inbound strictly from App Server only (Zero Trust Principle)
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.app_sg.id]
  }
}

variable "db_password" {
  description = "RDS Master Password"
  type        = string
  sensitive   = true
}
