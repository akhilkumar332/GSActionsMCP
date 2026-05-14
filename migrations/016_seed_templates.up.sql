INSERT INTO templates (name, description, config, is_public, is_premium) VALUES 
('Daily Standup Summarizer', 'Automatically gathers updates from team Slack channels and generates a structured daily summary.', '{"task_type": "mcp_sampling", "agent_prompt": "Read the #standup channel from the last 24 hours. Summarize blockers and progress by team member.", "trigger_type": "cron", "trigger_config": {"cron": "0 9 * * 1-5"}}', true, false),

('Weekly GitHub PR Review', 'Analyzes merged PRs from the past week and drafts release notes.', '{"task_type": "mcp_sampling", "agent_prompt": "Fetch merged PRs from the past 7 days on the main repository. Draft a release notes markdown document categorized by Features, Fixes, and Chores.", "trigger_type": "cron", "trigger_config": {"cron": "0 17 * * 5"}}', true, false),

('Customer Support Sentiment Analyzer', 'Triggered by a webhook to analyze incoming support tickets for urgency and user sentiment.', '{"task_type": "mcp_sampling", "agent_prompt": "Analyze the incoming ticket text. Determine the sentiment (Positive, Neutral, Negative) and urgency (Low, Medium, High). Format the output as JSON.", "trigger_type": "webhook", "trigger_config": {}}', true, true),

('Website Health Monitor', 'Pings an array of URLs every 15 minutes and alerts via Slack if any return non-200 status codes.', '{"task_type": "native_action", "native_code": "const urls = [''https://example.com''];\nfor(const url of urls) {\n  const res = await fetch(url);\n  if(!res.ok) console.error(`Alert: ${url} is down!`);\n}", "trigger_type": "interval", "trigger_config": {"minutes": 15}}', true, false),

('SEO Keyword Rank Tracker', 'Fetches weekly SEO rankings for specified keywords and updates a reporting dashboard.', '{"task_type": "mcp_sampling", "agent_prompt": "Check the current Google search rank for our target keywords. Output a CSV formatted table of Keyword, Rank, and Weekly Change.", "trigger_type": "cron", "trigger_config": {"cron": "0 8 * * 1"}}', true, true),

('Monthly Invoice Dispatch', 'Generates PDF invoices from billing data and emails them to clients.', '{"task_type": "mcp_sampling", "agent_prompt": "Fetch billing usage for all active clients for the previous month. Generate a professional invoice and dispatch via the email integration.", "trigger_type": "cron", "trigger_config": {"cron": "0 8 1 * *"}}', true, true),

('Social Media Content Scheduler', 'Drafts trending tweets based on industry news every few hours.', '{"task_type": "mcp_sampling", "agent_prompt": "Scan industry news sources for the latest AI trends. Draft 3 engaging tweets, including relevant hashtags, and schedule them.", "trigger_type": "interval", "trigger_config": {"minutes": 240}}', true, false),

('Stale Branch Cleanup', 'Finds and deletes stale git branches older than 30 days that have been merged.', '{"task_type": "mcp_sampling", "agent_prompt": "List all git branches that have been merged into main and have had no activity for 30 days. Delete them to keep the repository clean.", "trigger_type": "cron", "trigger_config": {"cron": "0 0 1 * *"}}', true, false),

('Competitor Pricing Tracker', 'Scrapes competitor websites daily to monitor pricing changes.', '{"task_type": "mcp_sampling", "agent_prompt": "Visit the pricing pages of listed competitors. Extract the current tier prices and compare them against our database. Alert if there are any price drops.", "trigger_type": "cron", "trigger_config": {"cron": "0 6 * * *"}}', true, true),

('New User Onboarding Email', 'Sends a personalized welcome sequence when a new user signs up.', '{"task_type": "mcp_sampling", "agent_prompt": "Take the user profile data from the webhook payload. Draft a highly personalized welcome email highlighting features relevant to their industry.", "trigger_type": "webhook", "trigger_config": {}}', true, false),

('Cloud Cost Anomaly Alert', 'Checks AWS/GCP billing daily and alerts if usage spikes significantly.', '{"task_type": "mcp_sampling", "agent_prompt": "Fetch the trailing 24-hour cloud spend. Compare it to the 7-day moving average. If the spend is 15% higher than average, trigger a PagerDuty alert.", "trigger_type": "cron", "trigger_config": {"cron": "0 7 * * *"}}', true, true),

('Meeting Notes Extractor', 'Processes raw transcript text into structured action items.', '{"task_type": "mcp_sampling", "agent_prompt": "Read the provided meeting transcript. Extract the key decisions made and create a bulleted list of action items, assigning them to the mentioned individuals.", "trigger_type": "webhook", "trigger_config": {}}', true, false),

('Error Log Anomaly Detector', 'Scans logs for abnormal error spikes every hour.', '{"task_type": "mcp_sampling", "agent_prompt": "Query the log aggregator for the last 60 minutes. Count the frequency of HTTP 5xx errors. If the count exceeds the threshold of 50, escalate to the engineering channel.", "trigger_type": "interval", "trigger_config": {"minutes": 60}}', true, true),

('HackerNews Top 10 Digest', 'Scrapes HackerNews and sends the top 10 articles to a Telegram channel.', '{"task_type": "mcp_sampling", "agent_prompt": "Fetch the top 10 current stories from HackerNews. Create a brief 1-sentence summary for each and format it nicely for Telegram.", "trigger_type": "cron", "trigger_config": {"cron": "0 18 * * *"}}', true, false),

('Jira Stale Issue Nag', 'Pings assignees on Jira issues untouched for over 14 days.', '{"task_type": "mcp_sampling", "agent_prompt": "Find all Jira issues in the ''In Progress'' state that haven''t been updated in 14 days. Add a comment tagging the assignee asking for a status update.", "trigger_type": "cron", "trigger_config": {"cron": "0 9 * * 3"}}', true, false),

('Newsletter Draft Generator', 'Compiles weekly company updates into a draft newsletter.', '{"task_type": "mcp_sampling", "agent_prompt": "Gather top performing blog posts, new feature releases, and community highlights from the past week. Draft a cohesive newsletter in HTML format.", "trigger_type": "cron", "trigger_config": {"cron": "0 10 * * 4"}}', true, false),

('CRM Lead Enrichment', 'Enriches a new email lead with LinkedIn and Clearbit data.', '{"task_type": "mcp_sampling", "agent_prompt": "Extract the domain from the provided email address. Query Clearbit and LinkedIn to find the company size, industry, and the lead''s job title. Update the CRM record.", "trigger_type": "webhook", "trigger_config": {}}', true, true),

('Daily Weather Briefing', 'Sends a concise weather and commute briefing every morning.', '{"task_type": "native_action", "native_code": "const weather = await fetch(''https://api.weather.gov/...'');\\n// Process and send email", "trigger_type": "cron", "trigger_config": {"cron": "30 7 * * *"}}', true, false);