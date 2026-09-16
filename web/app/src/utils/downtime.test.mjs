import { test } from 'node:test'
import assert from 'node:assert/strict'
import { downtimeIntervals, pendingIntervals } from './downtime.js'

const MINUTE = 60 * 1000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const NOW = Date.parse('2026-09-16T12:00:00Z')
const iso = (time) => new Date(time).toISOString()

test('a downtime of 10 minutes', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - 10 * HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - 10 * HOUR) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 2 * HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - 2 * HOUR + 10 * MINUTE) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW), [{ start: NOW - 2 * HOUR, end: NOW - 2 * HOUR + 10 * MINUTE }])
})

test('a downtime still in progress goes until the end of the period', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - 5 * HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - 5 * HOUR) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 3 * HOUR) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW), [{ start: NOW - 3 * HOUR, end: NOW }])
})

test('a downtime started before the period is clipped to it', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - 20 * DAY) },
    { type: 'HEALTHY', timestamp: iso(NOW - 20 * DAY) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 8 * DAY) },
    { type: 'HEALTHY', timestamp: iso(NOW - 6 * DAY) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - 7 * DAY, NOW), [{ start: NOW - 7 * DAY, end: NOW - 6 * DAY }])
})

test('a new endpoint without downtime has no period', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - HOUR) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW, NOW - HOUR), [])
})

test('a truncated list of events starting with HEALTHY closes a period from the first point of the chart', () => {
  const events = [
    { type: 'HEALTHY', timestamp: iso(NOW - 2 * HOUR) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - 30 * MINUTE) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW, NOW - 5 * HOUR), [
    { start: NOW - 5 * HOUR, end: NOW - 2 * HOUR },
    { start: NOW - HOUR, end: NOW - 30 * MINUTE },
  ])
  // Without point in the chart, or with a point older than the period, the period starts at the start of the period
  assert.deepEqual(downtimeIntervals(events.slice(0, 1), NOW - DAY, NOW), [{ start: NOW - DAY, end: NOW - 2 * HOUR }])
  assert.deepEqual(downtimeIntervals(events.slice(0, 1), NOW - DAY, NOW, NOW - 3 * DAY), [{ start: NOW - DAY, end: NOW - 2 * HOUR }])
  // A first point after the HEALTHY event gives no period
  assert.deepEqual(downtimeIntervals(events.slice(0, 1), NOW - DAY, NOW, NOW - HOUR), [])
})

test('a HEALTHY event without UNHEALTHY before it closes no period when the events have a START', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - 3 * HOUR) },
    { type: 'HEALTHY', timestamp: iso(NOW - 2 * HOUR) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW, NOW - 5 * HOUR), [])
})

test('a downtime that ended before the period is left out', () => {
  const events = [
    { type: 'START', timestamp: iso(NOW - 20 * DAY) },
    { type: 'HEALTHY', timestamp: iso(NOW - 20 * DAY) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 10 * DAY) },
    { type: 'HEALTHY', timestamp: iso(NOW - 10 * DAY + HOUR) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - 7 * DAY, NOW), [])
})

test('unordered events and repeated UNHEALTHY events give a single period', () => {
  const events = [
    { type: 'HEALTHY', timestamp: iso(NOW - HOUR) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 2 * HOUR) },
    { type: 'START', timestamp: iso(NOW - 4 * HOUR) },
    { type: 'UNHEALTHY', timestamp: iso(NOW - 90 * MINUTE) },
  ]
  assert.deepEqual(downtimeIntervals(events, NOW - DAY, NOW), [{ start: NOW - 2 * HOUR, end: NOW - HOUR }])
  assert.deepEqual(downtimeIntervals([], NOW - DAY, NOW), [])
  assert.deepEqual(downtimeIntervals(undefined, NOW - DAY, NOW), [])
})

test('pending results make yellow periods until the next result that is not pending', () => {
  const at = (minutes) => new Date(Date.UTC(2026, 8, 16, 10, minutes)).toISOString()
  const start = Date.UTC(2026, 8, 16, 9, 0)
  const end = Date.UTC(2026, 8, 16, 11, 0)
  const results = [
    { timestamp: at(0), success: true },
    { timestamp: at(5), pending: true },
    { timestamp: at(6), pending: true },
    { timestamp: at(7), success: false },
    { timestamp: at(30), pending: true },
  ]
  assert.deepEqual(pendingIntervals(results, start, end), [
    { start: Date.UTC(2026, 8, 16, 10, 5), end: Date.UTC(2026, 8, 16, 10, 7) },
    { start: Date.UTC(2026, 8, 16, 10, 30), end },
  ])
  assert.deepEqual(pendingIntervals([{ timestamp: at(0), success: true }], start, end), [])
  assert.deepEqual(pendingIntervals(results, Date.UTC(2026, 8, 16, 10, 6), Date.UTC(2026, 8, 16, 10, 20)), [
    { start: Date.UTC(2026, 8, 16, 10, 6), end: Date.UTC(2026, 8, 16, 10, 7) },
  ])
})
