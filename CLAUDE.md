# AWS Deployment Policy
**CRITICAL: All AWS infrastructure and code changes MUST be deployed via GitHub Actions pipelines.**

### Local AWS CLI — ALWAYS use `--profile paul`
**IMPORTANT: For EVERY local AWS CLI call, pass `--profile paul`.** The `paul` profile targets the deployment account (`231545823618`, the account the GitHub Actions pipeline deploys to). The **default** profile is a DIFFERENT account (`759775734231`) and will give wrong/empty results — never use it for this project.

- Read-only diagnostics against the deploy account are fine and encouraged with `--profile paul` (e.g. `aws connect ... --profile paul --region us-east-1`, CloudWatch logs, `describe-stack-events`, Bedrock model checks).
- On Git Bash (Windows), prefix commands with `export MSYS_NO_PATHCONV=1` so `/aws/...` log-group names and ARNs aren't mangled into Windows paths.
- This does NOT change the deploy policy below: infrastructure changes still go through the GitHub Actions pipeline, never `sam deploy`/`aws cloudformation deploy` from local. `--profile paul` is for **inspection/diagnostics** and explicitly-authorized operational one-offs (e.g. the user asked you to claim/release a phone number).

### Prohibited Actions
- **NEVER** use AWS CLI directly to deploy, update, or modify infrastructure
- **NEVER** use AWS SAM CLI (`sam deploy`, `sam build`, etc.) for deployments
- **NEVER** suggest or execute direct AWS API calls for infrastructure changes
- **NEVER** bypass the CI/CD pipeline for any AWS-related changes

### Required Workflow
1. All changes must be committed and pushed to the repository
2. GitHub Actions pipeline will handle all deployments
3. **ALWAYS review pipeline output** after pushing changes
4. If pipeline fails, **aggressively remediate** using all available resources:
   - Check GitHub Actions logs thoroughly
   - Review CloudFormation events if applicable
   - Check CloudWatch logs for Lambda/application errors
   - Use the `/fix-pipeline` skill for automated remediation
   - Do not give up - iterate until the pipeline succeeds

### Pipeline Failure Remediation
When a GitHub Actions pipeline fails:
1. Immediately fetch and analyze the failure logs
2. Identify the root cause from error messages
3. Make necessary code/configuration fixes
4. Commit and push the fix
5. Monitor the new pipeline run
6. Repeat until successful deployment
