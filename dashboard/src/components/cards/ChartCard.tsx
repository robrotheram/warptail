import { ResponsiveContainer, CartesianGrid, XAxis, YAxis, Tooltip, Legend, Line, LineChart } from "recharts"
import { Card, CardHeader, CardTitle, CardContent } from "../ui/card"
import { formatBytes, formatXAxis, getLast10MinutesData } from "../../lib/utils"
import { Service } from "@/lib/api"

type RouterChartProps = {
  service: Service
}
export const RouterChart = ({service}:RouterChartProps) => {
  const timeSeries = getLast10MinutesData(service.stats.points)
  return <Card>
  <CardHeader className="py-3">
    <CardTitle className="text-lg">Traffic Statistics</CardTitle>
  </CardHeader>
  <CardContent className="space-y-2">
    <div className="flex justify-between">
      <p>
        <strong>Total Sent:</strong> {formatBytes(service.stats.total.sent)}
      </p>
      <p>
        <strong>Total Received:</strong>{' '}
        {formatBytes(service.stats.total.received)}
      </p>
    </div>
    {timeSeries.length > 0 &&
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
    }
  </CardContent>
</Card>
}