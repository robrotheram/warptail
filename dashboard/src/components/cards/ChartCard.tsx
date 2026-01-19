import { useMemo, useState } from "react"
import { ResponsiveContainer, CartesianGrid, XAxis, YAxis, Tooltip, Legend, Line, LineChart } from "recharts"
import { Card, CardHeader, CardTitle, CardContent } from "../ui/card"
import { formatBytes, formatXAxis, getLast10MinutesData } from "../../lib/utils"
import { ProxyStats, Route, Service, TimeSeries, TimeSeriesPoint } from "@/lib/api"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select"

const SUMMARY_VALUE = "__summary__"

/**
 * Get a display label for a route
 */
function getRouteLabel(route: Route, index: number): string {
  if (route.domain) {
    return route.domain
  }
  if (route.port) {
    return `${route.type?.toUpperCase() || 'TCP'}:${route.port}`
  }
  return `Route ${index + 1}`
}

/**
 * Combine multiple TimeSeries into a single aggregated TimeSeries
 */
function aggregateTimeSeries(routes: Route[]): TimeSeries {
  const routesWithStats = routes.filter(r => r.stats && r.stats.points?.length > 0)
  
  if (routesWithStats.length === 0) {
    return { points: [], total: { sent: 0, received: 0 } }
  }

  // Aggregate totals
  const total: ProxyStats = routesWithStats.reduce(
    (acc, route) => ({
      sent: acc.sent + (route.stats?.total?.sent || 0),
      received: acc.received + (route.stats?.total?.received || 0),
    }),
    { sent: 0, received: 0 }
  )

  // Combine all points into a map by timestamp
  const pointsMap = new Map<string, ProxyStats>()
  
  for (const route of routesWithStats) {
    if (!route.stats?.points) continue
    for (const point of route.stats.points) {
      const key = new Date(point.timestamp).toISOString()
      const existing = pointsMap.get(key) || { sent: 0, received: 0 }
      pointsMap.set(key, {
        sent: existing.sent + point.value.sent,
        received: existing.received + point.value.received,
      })
    }
  }

  // Convert map back to sorted array
  const points: TimeSeriesPoint[] = Array.from(pointsMap.entries())
    .sort(([a], [b]) => new Date(a).getTime() - new Date(b).getTime())
    .map(([timestamp, value]) => ({
      timestamp: new Date(timestamp),
      value,
    }))

  return { points, total }
}

type RouterChartProps = {
  service: Service
}

export const RouterChart = ({ service }: RouterChartProps) => {
  const [selectedRoute, setSelectedRoute] = useState<string>(SUMMARY_VALUE)

  // Build route options for the filter
  const routeOptions = useMemo(() => {
    return service.routes.map((route, index) => ({
      value: String(route.key ?? index),
      label: getRouteLabel(route, index),
      route,
    }))
  }, [service.routes])

  // Get the stats based on selection
  const { stats, label } = useMemo(() => {
    if (selectedRoute === SUMMARY_VALUE) {
      return {
        stats: aggregateTimeSeries(service.routes),
        label: "All Routes (Summary)",
      }
    }

    const routeIndex = parseInt(selectedRoute, 10)
    const route = service.routes.find((r, i) => (r.key ?? i) === routeIndex)
    
    if (!route || !route.stats) {
      return {
        stats: { points: [], total: { sent: 0, received: 0 } },
        label: "No Data",
      }
    }

    return {
      stats: route.stats,
      label: getRouteLabel(route, routeIndex),
    }
  }, [selectedRoute, service.routes])

  const timeSeries = useMemo(() => {
    if (!stats.points || stats.points.length === 0) return []
    return getLast10MinutesData(stats.points)
  }, [stats.points])

  const hasMultipleRoutes = service.routes.length > 1

  return (
    <Card>
      <CardHeader className="py-3 flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-lg">Traffic Statistics</CardTitle>
        {hasMultipleRoutes && (
          <Select value={selectedRoute} onValueChange={setSelectedRoute}>
            <SelectTrigger className="w-[200px]">
              <SelectValue placeholder="Select route" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={SUMMARY_VALUE}>All Routes (Summary)</SelectItem>
              {routeOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="flex justify-between">
          <p>
            <strong>Total Sent:</strong> {formatBytes(stats.total?.sent || 0)}
          </p>
          <p>
            <strong>Total Received:</strong>{' '}
            {formatBytes(stats.total?.received || 0)}
          </p>
        </div>
        {timeSeries.length > 0 ? (
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={timeSeries}>
              <CartesianGrid strokeDasharray="2" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatXAxis}
                domain={['dataMin', 'dataMax']}
              />
              <YAxis tickFormatter={formatBytes} />
              <Tooltip
                contentStyle={{ backgroundColor: '#000' }}
                formatter={formatBytes}
                labelFormatter={formatXAxis}
              />
              <Legend />
              <Line
                type="monotone"
                dataKey="value.sent"
                stroke="#8884d8"
                name="sent"
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="value.received"
                stroke="#82ca9d"
                name="received"
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex items-center justify-center h-[250px] text-muted-foreground">
            No traffic data available
          </div>
        )}
      </CardContent>
    </Card>
  )
}