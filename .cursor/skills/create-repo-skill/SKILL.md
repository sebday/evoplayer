---
name: create-repo-skill
description: Create or update a project skill in .cursor/skills. Use when adding a new skill, converting a workflow into a skill, or reviewing skill frontmatter and descriptions.
---

# Create a repo skill

## Layout

Project skills live at `.cursor/skills/<kebab-name>/SKILL.md`. Cursor auto-discovers them from there.

Update the skill roster in `AGENTS.md` when adding a skill.

## Creating a skill

1. `mkdir .cursor/skills/<kebab-name>` and write `SKILL.md` with frontmatter:

   ```markdown
   ---
   name: <kebab-name>            # must match the directory name
   description: <what it does + explicit "use when ..." triggers>
   ---
   ```

2. Add a one-line entry under **Skills** in `AGENTS.md` describing when to load it.

The `description` field is load-bearing — Cursor uses it to decide when to apply the skill. Include concrete trigger phrases.
