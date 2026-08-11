<script setup lang="ts">
defineProps<{ title: string; subtitle: string }>();
</script>
<template>
  <div class="auth-page">
    <div class="ambient-canvas" aria-hidden="true">
      <span class="ambient-blob ambient-blob-primary"></span>
      <span class="ambient-blob ambient-blob-left"></span>
      <span class="ambient-blob ambient-blob-right"></span>
    </div>
    <section v-spotlight class="auth-card">
      <router-link to="/" class="auth-brand"
        ><span class="brand-mark"><span></span></span>TokenPulse</router-link
      >
      <div class="auth-heading">
        <span class="eyebrow">AI USAGE INTELLIGENCE</span>
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
      </div>
      <slot />
    </section>
    <aside class="auth-visual">
      <div class="visual-copy">
        <span class="live-label"><span class="pulse-dot"></span>LIVE TOKEN FLOW</span>
        <strong>3,520,100</strong>
        <p>在所有设备间，保持每一次 AI 协作清晰可见。</p>
        <div class="flow-visual" aria-hidden="true">
          <span v-for="index in 24" :key="index" :style="{ '--bar-index': index }"></span>
        </div>
      </div>
    </aside>
  </div>
</template>
