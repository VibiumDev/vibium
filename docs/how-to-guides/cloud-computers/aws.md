# Browser server on AWS

The most setup friction of the platforms here. Use this when you already live in
AWS (spot instances make it very cheap); use Fly or DigitalOcean when
you don't.

## Which AWS shape?

"AWS" is several different answers:

| Shape | When | Notes |
|---|---|---|
| **EC2 on-demand** (this recipe) | Default; steady use | t3.medium runs 2–3 headless Chromes |
| **EC2 Spot** | Disposable/burst boxes | Same recipe, `--instance-market-options`; interruptible by design |
| **Graviton (arm64)** | When Chrome for Testing ships linux-arm64 | Cheaper per vCPU; today needs a Chromium build instead — watch the CfT feed |
| **Fargate** | Container workflow, no instance management | Per-vCPU-second; use the Fly recipe's Dockerfile; no KVM, slower cold start |
| Lambda | — | Don't: execution caps and cold starts fight the browser-session model |

AWS's managed AgentCore Browser is the managed sibling — a hosted
browser API rather than a box you own, and it speaks CDP, not WebDriver
BiDi, so vibium can't drive it today.

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
  --user-data file://deploy/cloud-computers/aws/user-data.sh \
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
disposable machines.
