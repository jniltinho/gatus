// Periods out of service of the response time chart, computed from the events of the endpoint (fork)

const toMilliseconds = (value) => (typeof value === 'number' ? value : new Date(value).getTime())

// downtimeIntervals returns the periods out of service ({start, end}, in milliseconds) that overlap the period between
// periodStart and periodEnd, clipped to it. A period goes from an UNHEALTHY event to the next HEALTHY event, or to the
// end of the period when the endpoint is still out of service. The first HEALTHY event, without UNHEALTHY before it,
// only closes a period when the events have no START, which means the list was truncated: the period then starts at
// the latest of the start of the period and the first point of the chart (firstPointTime, optional). START followed by
// HEALTHY is the normal start of the monitoring, without period.
export const downtimeIntervals = (events, periodStart, periodEnd, firstPointTime = null) => {
  const from = toMilliseconds(periodStart)
  const to = toMilliseconds(periodEnd)
  const sorted = (events || [])
    .filter((event) => ['START', 'HEALTHY', 'UNHEALTHY'].includes(event.type))
    .map((event) => ({ type: event.type, time: toMilliseconds(event.timestamp) }))
    .filter((event) => Number.isFinite(event.time))
    .sort((a, b) => a.time - b.time)
  const truncated = !sorted.some((event) => event.type === 'START')
  const intervals = []
  let openedAt = null
  let transitionSeen = false
  for (const event of sorted) {
    if (event.type === 'UNHEALTHY') {
      if (openedAt === null) {
        openedAt = event.time
      }
      transitionSeen = true
    } else if (event.type === 'HEALTHY') {
      if (openedAt !== null) {
        intervals.push({ start: openedAt, end: event.time })
        openedAt = null
      } else if (!transitionSeen && truncated) {
        const firstPoint = firstPointTime === null || firstPointTime === undefined ? from : toMilliseconds(firstPointTime)
        intervals.push({ start: Math.max(from, firstPoint), end: event.time })
      }
      transitionSeen = true
    }
  }
  if (openedAt !== null) {
    intervals.push({ start: openedAt, end: to })
  }
  return intervals
    .map((interval) => ({ start: Math.max(interval.start, from), end: Math.min(interval.end, to) }))
    .filter((interval) => interval.end > interval.start)
}

// pendingIntervals returns the periods of consecutive pending results ({start, end}, in milliseconds) that overlap the
// period between periodStart and periodEnd, clipped to it (fork). A period goes from the first pending result to the
// next result that is not pending, or to the end of the period when the last result is pending. Pending results do
// not create events, so the periods only cover the results received by the chart.
export const pendingIntervals = (results, periodStart, periodEnd) => {
  const from = toMilliseconds(periodStart)
  const to = toMilliseconds(periodEnd)
  const sorted = (results || [])
    .map((result) => ({ pending: result.pending === true, time: toMilliseconds(result.timestamp) }))
    .filter((result) => Number.isFinite(result.time))
    .sort((a, b) => a.time - b.time)
  const intervals = []
  let openedAt = null
  for (const result of sorted) {
    if (result.pending) {
      if (openedAt === null) {
        openedAt = result.time
      }
    } else if (openedAt !== null) {
      intervals.push({ start: openedAt, end: result.time })
      openedAt = null
    }
  }
  if (openedAt !== null) {
    intervals.push({ start: openedAt, end: to })
  }
  return intervals
    .map((interval) => ({ start: Math.max(interval.start, from), end: Math.min(interval.end, to) }))
    .filter((interval) => interval.end > interval.start)
}
