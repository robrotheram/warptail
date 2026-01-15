import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"
import { ProxyStats, TimeSeriesPoint } from "./api"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}


export const formatXAxis = (tickItem: string) => {
  const date = new Date(tickItem)
  const hours = date.getHours().toString().padStart(2, '0')
  const minutes = date.getMinutes().toString().padStart(2, '0')
  const seconds = date.getSeconds().toString().padStart(2, '0');
  return `${hours}:${minutes}:${seconds}`
}

export const formatBytes = (bytes: number): string =>{
  if (bytes < 1024) return `${bytes.toFixed(0)} B`
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const formattedSize = (bytes / Math.pow(1024, i)).toFixed(0)
  return `${formattedSize} ${sizes[i]}`
}

export const formatDuration = (nanoseconds?: number): string => {
  if (nanoseconds === undefined || nanoseconds < 0) return '-'
  if (nanoseconds === 0) return '0 ns'
  
  const units = [
    { label: 's', value: 1_000_000_000 },
    { label: 'ms', value: 1_000_000 },
    { label: 'µs', value: 1_000 },
    { label: 'ns', value: 1 },
  ]
  
  for (const unit of units) {
    if (nanoseconds >= unit.value) {
      const value = nanoseconds / unit.value
      return `${value.toFixed(value < 10 ? 2 : value < 100 ? 1 : 0)} ${unit.label}`
    }
  }
  
  return `${nanoseconds} ns`
}

/**
 * Function to interpolate missing ProxyStats values between two data points.
 * @param prev The previous data point.
 * @param next The next data point.
 * @param fraction The fraction of time between the two points.
 */
function interpolateProxyStats(prev: ProxyStats, next: ProxyStats, fraction: number): ProxyStats {
  return {
      sent: prev.sent + (next.sent - prev.sent) * fraction,
      received: prev.received + (next.received - prev.received) * fraction
  };
}

/**
 * Function to apply a moving average to smooth out spikes in the data.
 * @param points The input array of Points.
 * @param windowSize The number of points to average over.
 */
function applyMovingAverage(points: TimeSeriesPoint[], windowSize: number): TimeSeriesPoint[] {
  const smoothedPoints: TimeSeriesPoint[] = [];
  for (let i = 0; i < points.length; i++) {
      let avgSent = 0;
      let avgReceived = 0;
      let count = 0;

      // Calculate the average over the window
      for (let j = Math.max(0, i - windowSize + 1); j <= i; j++) {
          avgSent += points[j].value.sent;
          avgReceived += points[j].value.received;
          count++;
      }

      // Add the averaged point
      smoothedPoints.push({
          timestamp: points[i].timestamp,
          value: {
              sent: avgSent / count,
              received: avgReceived / count
          }
      });
  }
  return smoothedPoints;
}

/**
* Generates a new set of Points containing the last 10 minutes of data with interpolated values for missing points.
* @param points The input array of Points (with Timestamp and ProxyStats values).
*/
export function getLast10MinutesData(points: TimeSeriesPoint[]): TimeSeriesPoint[] {
  if (points.length === 0) return [];

  points.forEach(point => {point.timestamp = new Date(point.timestamp)});
  const now = new Date();
  const tenMinutesAgo = new Date(now.getTime() - 10 * 60 * 1000); // Current time minus 10 minutes
  const filteredPoints = points.filter(point => point.timestamp >= tenMinutesAgo);
  
  if (filteredPoints.length === 0) return [];

  const smoothPoints: TimeSeriesPoint[] = [];

  smoothPoints.push(filteredPoints[0]);
  const interval = 60 * 1000;

  for (let i = 1; i < filteredPoints.length; i++) {
      const prevPoint = filteredPoints[i - 1];
      const currentPoint = filteredPoints[i];
      const timeDiff = currentPoint.timestamp.getTime() - prevPoint.timestamp.getTime();
      if (timeDiff > interval) {
          const numMissingPoints = Math.floor(timeDiff / interval);

          for (let j = 1; j <= numMissingPoints; j++) {
              const interpolatedTimestamp = new Date(prevPoint.timestamp.getTime() + j * interval);
              const fraction = j / numMissingPoints; // Fraction of how far between the points we are
              const interpolatedValue = interpolateProxyStats(prevPoint.value, currentPoint.value, fraction);

              smoothPoints.push({
                timestamp: interpolatedTimestamp,
                  value: interpolatedValue
              });
          }
      }
      smoothPoints.push(currentPoint);
  }

  return applyMovingAverage(smoothPoints, 3);
}


export type ParsedLog = {
  timestamp: string;
  level: string;
  message: string;
};


export const parseLog = (log: string): ParsedLog | null => {
  console.log(log);
  // log = `2025-04-11T20:49:40.236+0100\\t\\u001b[34mINFO\\u001b[0m\\tAuthLoop: state is Running; done`;
  const regex = /^(.+?)\\t\\u001b\[.*?m(.*?)\\u001b\[0m\\t(.+)$/;
  const match = regex.exec(log);

  if (!match) return null;

  const [, rawTimestamp, level, message] = match;

  const timestamp = (rawTimestamp);

  return {
    timestamp,
    level,
    message,
  };
}


