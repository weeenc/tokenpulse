<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import { Monitor, CircleCheckFilled, WarningFilled } from '@element-plus/icons-vue';
import { api, data, errorMessage } from '../api/client.js';
import { approvalPayload, platformName, type DeviceChoice } from '../utils/device-auth.js';

interface Device {
  id: number;
  deviceId: string;
  deviceName: string;
  platform: string;
  arch: string;
  lastActiveAt?: string;
}
interface AuthInfo {
  request: {
    userCode: string;
    deviceName: string;
    platform: string;
    arch: string;
    agentVersion: string;
    expiresAt: string;
  };
  devices: Device[];
}
const route = useRoute();
const info = ref<AuthInfo>();
const choice = ref<DeviceChoice>('new');
const loading = ref(true);
const finished = ref<'approved' | 'denied'>();
const code = computed(() => String(route.query.code ?? '').toUpperCase());
onMounted(load);
async function load() {
  if (!code.value) {
    loading.value = false;
    return;
  }
  try {
    info.value = await data<AuthInfo>(
      api.get(`/device-auth/info/${encodeURIComponent(code.value)}`),
    );
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}
async function approve() {
  try {
    await api.post('/device-auth/approve', approvalPayload(code.value, choice.value));
    finished.value = 'approved';
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}
async function deny() {
  try {
    await api.post('/device-auth/deny', { userCode: code.value });
    finished.value = 'denied';
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}
</script>
<template>
  <div class="device-auth-page">
    <section v-loading="loading" v-spotlight class="device-auth-card">
      <template v-if="finished"
        ><el-icon class="result-icon" :class="finished"
          ><CircleCheckFilled v-if="finished === 'approved'" /><WarningFilled v-else
        /></el-icon>
        <h1>{{ finished === 'approved' ? '设备已连接' : '已拒绝授权' }}</h1>
        <p>
          {{
            finished === 'approved'
              ? '你可以关闭此页面并返回终端。'
              : '此设备不会访问你的 TokenPulse 账户。'
          }}
        </p></template
      ><template v-else-if="info"
        ><span class="eyebrow">DEVICE AUTHORIZATION</span>
        <div class="device-symbol">
          <el-icon><Monitor /></el-icon>
        </div>
        <h1>设备请求授权</h1>
        <p class="muted">确认这是你正在使用的设备。TokenPulse 只同步 Token 使用元数据。</p>
        <div class="request-device">
          <strong>{{ info.request.deviceName }}</strong
          ><span>{{ platformName(info.request.platform) }} · {{ info.request.arch }}</span
          ><small>TokenPulse Agent {{ info.request.agentVersion }}</small>
        </div>
        <div class="choice-list">
          <label :class="{ selected: choice === 'new' }"
            ><input v-model="choice" type="radio" value="new" /><span
              ><strong>添加为新设备</strong><small>创建独立的设备记录</small></span
            ></label
          ><label
            v-for="device in info.devices"
            :key="device.id"
            :class="{ selected: choice === device.id }"
            ><input v-model="choice" type="radio" :value="device.id" /><span
              ><strong>重新连接 {{ device.deviceName }}</strong
              ><small>保留历史数据并轮换旧凭证</small></span
            ></label
          >
        </div>
        <div class="device-actions">
          <el-button size="large" @click="deny">拒绝</el-button
          ><el-button type="primary" size="large" @click="approve">允许连接</el-button>
        </div></template
      ><template v-else
        ><h1>授权代码无效</h1>
        <p>请返回终端重新运行 tokenpulse login。</p></template
      >
    </section>
  </div>
</template>
