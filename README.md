# NGINX Cache Sync Helper

A lightweight Go-based background daemon that syncs NGINX FastCGI/Proxy cache clears across **multiple WordPress replicas**.  
Designed for clustered or containerized environments where **each replica maintains its own local NGINX cache**, this tool ensures consistent cache invalidation by polling a shared MySQL database for updates.

---

## Features

- Polls central WordPress `wp_options` table for `nginx_cache_last_cleared`
- Compares to local state to determine if cache needs clearing
- Clears local NGINX cache when needed
- Stores last-seen timestamp in a local file
---

## WordPress Integration

Use this together with the companion plugin:

### [`pds-uk/nginx-cache-helper`](https://github.com/pds-uk/nginx-cache-helper)

This WordPress plugin:

- Automatically clears the local NGINX cache on post, comment, plugin, or WooCommerce events
- Updates the `nginx_cache_last_cleared` value in the WordPress database
- Supports Composer installation:

```bash
composer require pds-uk/nginx-cache-helper