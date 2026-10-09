---
title: device builder silently skips endpoints
state: researching
created: 2026-10-07
tags: [defect]
source: patchbay stage 7
---

`topology.DeviceBuilder` skips a routing endpoint when its control is missing or its first read fails, and appends the rest, so `Device.PCMCaptureEndpoints[N]` is not necessarily PCM capture channel N; a consumer indexing by position lands on the wrong channel. either keep the slice positional with a nil for a skipped channel, or make the skip loud (an error, or a recorded gap on the Device) and say in the type's doc that the slice is positional or that it is not. patchbay works around it by looking endpoints up by port number.
