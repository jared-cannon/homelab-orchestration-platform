# Intelligent Orchestration

## Overview

Multi-node orchestration system that provides automatic device selection, resource optimization, and unified management:

1. **Intelligent Placement** - Automatic optimal device selection based on resource availability
2. **Database Pooling** - Shared database instances to reduce resource consumption
3. **Resource Aggregation** - Unified view of all devices and resources

---

## 1. Intelligent Placement Algorithm

Automatically selects the optimal device for app deployment based on weighted scoring:

**Scoring Criteria:**
- RAM availability: **40%** - Most critical for Docker containers
- Storage availability: **30%** - Important for persistent data
- CPU capability: **15%** - Less critical for most homelab apps
- Current load: **10%** - Prefer less-loaded devices
- Uptime/reliability: **5%** - Prefer stable devices

**How it Works:**
1. User requests app deployment (e.g., "Deploy NextCloud")
2. System scores all available devices against app requirements
3. System recommends optimal device with reasoning
4. User confirms or selects alternative

Devices that don't meet minimum requirements are disqualified. Scoring scales from 0-100 for each criterion, then combines using the weights above.

---

## 2. Database Pooling

Reduces resource consumption by sharing database instances across applications.

**Traditional Approach:**
- Each app deploys its own database container
- 5 apps = 5 Postgres containers = ~5GB RAM

**Pooling Approach:**
- One shared Postgres container per device
- All apps use separate databases within that instance
- 5 apps = 1 Postgres container = ~1.5GB RAM (**70% reduction**)

**How it Works:**
1. Check if shared database instance exists on target device
2. If not, deploy shared container with secure master credentials
3. Create isolated database within shared instance for the app
4. Generate dedicated user credentials
5. Inject credentials into app environment variables

Apps specify database requirements in their manifest, and the system automatically provisions isolated databases in shared instances.

---

## 3. Resource Aggregation

Provides unified view of resources across all devices in the homelab.

**Displays:**
- Total and used RAM across all devices
- Total and used storage across all devices
- Average CPU usage across devices
- Number of online/offline devices
- Total apps and their statuses
- RAM savings from database pooling

**How it Works:**
1. Periodically polls all registered devices for current metrics
2. Aggregates resource data (RAM, storage, CPU)
3. Counts apps by status (running, stopped)
4. Calculates resource savings from shared database instances
5. Presents unified dashboard view

This creates the experience of managing a single system rather than multiple individual devices.

---

## Summary

These features combine to provide Kubernetes-level intelligence with single-node simplicity:

- **Intelligent Placement** - System automatically picks optimal device
- **Database Pooling** - 60-70% RAM reduction through shared instances
- **Resource Aggregation** - Manage entire homelab as unified system

**Competitive Advantage:**
- CasaOS: Single node only
- Coolify: Manual device selection, no pooling
- Proxmox: Manual allocation, complex UI
- Kubernetes: Similar features but requires extensive expertise

---

**Last Updated:** October 2025
