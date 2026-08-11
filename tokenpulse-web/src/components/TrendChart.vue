<script setup lang="ts">
import { init, use, type ECharts } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';

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
let chart: ECharts | null = null;
function render() {
  if (!root.value) return;
  chart ??= init(root.value);
  chart.setOption({
    animationDuration: 700,
    animationEasing: 'cubicOut',
    grid: { top: 34, right: 18, bottom: 28, left: 54 },
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(13,13,17,.94)',
      borderColor: 'rgba(255,255,255,.09)',
      textStyle: { color: '#d5d6dc', fontSize: 11 },
      extraCssText: 'box-shadow: 0 16px 44px rgba(0,0,0,.45); backdrop-filter: blur(14px);',
      axisPointer: { lineStyle: { color: 'rgba(132,144,239,.28)' } },
    },
    legend: {
      top: 0,
      right: 8,
      itemWidth: 12,
      itemHeight: 3,
      textStyle: { color: '#747983', fontSize: 10 },
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.points.map((p) => p.date.slice(5)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,.07)' } },
      axisLabel: { color: '#5f646e', fontSize: 9, margin: 12 },
    },
    yAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: 'rgba(255,255,255,.045)', type: 'dashed' } },
      axisLabel: { color: '#5f646e', fontSize: 9, formatter: (value: number) => compact(value) },
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
          color: '#7c87e8',
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
        lineStyle: { width: 1.5, color: '#62cba6' },
      },
      {
        name: '输出',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.outputTokens),
        lineStyle: { width: 1.5, color: '#d7a967' },
      },
      {
        name: '缓存',
        type: 'line',
        smooth: true,
        symbol: 'none',
        data: props.points.map((p) => p.cachedInputTokens),
        lineStyle: { width: 1.5, color: '#6c9fca' },
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
