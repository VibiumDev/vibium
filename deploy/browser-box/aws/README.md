# AWS browser box (EC2)

The most setup friction of the kits. Use this when you already live in
AWS (spot instances make it very cheap); use Fly or DigitalOcean when
you don't.

## Prereqs (human)

1. AWS account (card + phone verification; new accounts get ~$100 credit)
2. AWS CLI configured (`aws configure`)
3. A key pair and a security group allowing **only** SSH (22) from your IP
4. An Ubuntu 24.04 AMI id for your region
   (`aws ssm get-parameter --name /aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id`)

## Create

```bash
aws ec2 run-instances \
  --image-id <ami-id> --instance-type t3.medium \
  --key-name <keypair> --security-group-ids <ssh-only-sg> \
  --user-data file://deploy/browser-box/aws/user-data.sh \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=vibium-browser}]'
```

First boot takes ~3–5 min while user-data installs Chrome; bake an AMI
afterward for ~1 min starts.

## Connect

```bash
ssh -N -L 9515:127.0.0.1:9515 ubuntu@<public-ip> &
vibium start http://127.0.0.1:9515
vibium go https://example.com
vibium stop
```

## Teardown

```bash
aws ec2 terminate-instances --instance-ids <id>
```

Per-second billing. Spot instances cut the price ~55% and are fine for
disposable browser boxes.
