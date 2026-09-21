<script setup lang="ts">
import { init, use, type ECharts } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useTheme } from '../utils/theme.js';

use([LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

interface Point {
  date: string;
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cachedInputTokens: number;
}
const props = defineProps<{ points: Point[] }>();
const root = ref<HTMLDivElement>();
const { theme } = useTheme();
let chart: ECharts | null = null;
function render() {
  if (!root.value) return;
  const styles = getComputedStyle(root.value);
  const color = (name: string) => styles.getPropertyValue(name).trim();
  const colors = ['--chart-total', '--chart-input', '--chart-output', '--chart-cache'].map(color);
  chart ??= init(root.value);
  chart.setOption({
    color: colors,
    animationDuration: 700,
    animationEasing: 'cubicOut',
    grid: { top: 34, right: 18, bottom: 28, left: 54 },
    tooltip: {
      trigger: 'axis',
      backgroundColor: color('--overlay-background'),
      borderColor: color('--border-default'),
      textStyle: { color: color('--foreground'), fontSize: 11 },
      extraCssText: `box-shadow: ${color('--el-box-shadow-light')}; backdrop-filter: blur(14px);`,
      axisPointer: { lineStyle: { color: 'rgba(132,144,239,.28)' } },
    },
    legend: {
      top: 0,
      right: 8,
      itemWidth: 12,
      itemHeight: 3,
      textStyle: { color: color('--foreground-muted'), fontSize: 10 },
      inactiveColor: color('--foreground-muted'),
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.points.map((p) => p.date.slice(5)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: color('--border-default') } },
      axisLabel: { color: color('--foreground-muted'), fontSize: 9, margin: 12 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: color('--border-default'), type: 'dashed' } },
      axisLabel: {
        color: color('--foreground-muted'),
        fontSize: 9,
        formatter: (value: number) => compact(value),
      },
    },
    series: [
      {
        name: '总量',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.totalTokens),
        lineStyle: {
          width: 2.5,
          color: colors[0],
          shadowColor: 'rgba(94,106,210,.4)',
          shadowBlur: 12,
        },
        areaStyle: { color: 'rgba(94,106,210,.11)' },
      },
      {
        name: '输入',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.inputTokens),
        lineStyle: { width: 1.5, color: colors[1] },
      },
      {
        name: '输出',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.outputTokens),
        lineStyle: { width: 1.5, color: colors[2] },
      },
      {
        name: '缓存',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.cachedInputTokens),
        lineStyle: { width: 1.5, color: colors[3] },
      },
    ],
  });
}
const resize = () => chart?.resize();
onMounted(() => {
  render();
  window.addEventListener('resize', resize);
});
watch(() => props.points, render, { deep: true });
watch(theme, render, { flush: 'post' });
onBeforeUnmount(() => {
  window.removeEventListener('resize', resize);
  chart?.dispose();
});
function compact(value: number) {
  return Intl.NumberFormat('zh-CN', { notation: 'compact', maximumFractionDigits: 1 }).format(
    value,
  );
}
</script>
<template><div ref="root" class="trend-chart"></div></template>
