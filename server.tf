resource "aws_instance" "server" {
  ami                         = var.ami
  instance_type               = "t2.nano"
  subnet_id                   = aws_subnet.pub.id
  associate_public_ip_address = true
  vpc_security_group_ids      = [aws_security_group.web.id]
  key_name                    = var.key_name

  connection {
    host        = self.public_ip
    type        = "ssh"
    user        = "ec2-user"
    private_key = file(var.key_file)
  }

  provisioner "remote-exec" {
    inline = [
      "echo | sudo tee -a /etc/ssh/sshd_config",
      "echo 'Port ${var.ssh_port}' | sudo tee -a /etc/ssh/sshd_config",
      "sudo service sshd reload",
    ]
  }
}
