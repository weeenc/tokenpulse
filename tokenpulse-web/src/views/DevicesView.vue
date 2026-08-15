<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { Monitor, EditPen, CircleClose } from '@element-plus/icons-vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { api, data, errorMessage, isCanceledRequest } from '../api/client.js';
import { platformName } from '../utils/device-auth.js';
import { formatDateTime } from '../utils/format.js';
interface Device {
  id: number;
  deviceId: string;
  deviceName: string;
  platform: string;
  arch: string;
  status: string;
  agentVersion?: string;
  lastActiveAt?: string;
  createdAt: string;
}
const devices = ref<Device[]>([]);
const loading = ref(false);
async function load() {
  loading.value = true;
  try {
    devices.value = await data(api.get('/devices'));
  } catch (error) {
    if (!isCanceledRequest(error)) ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}
async function rename(device: Device) {
  try {
    const value = await ElMessageBox.prompt('输入新的设备名称', '重命名设备', {
      inputValue: device.deviceName,
      inputPattern: /^.{1,128}$/,
      inputErrorMessage: '请输入设备名称',
    });
    await api.patch(`/devices/${device.id}`, { deviceName: value.value });
    ElMessage.success('设备名称已更新');
    await load();
  } catch (error) {
    if (error !== 'cancel' && error !== 'close' && !isCanceledRequest(error))
      ElMessage.error(errorMessage(error));
  }
}
async function revoke(device: Device) {
  try {
    await ElMessageBox.confirm(
      `撤销 ${device.deviceName} 后，该设备需要重新登录。历史 Token 数据会保留。`,
      '撤销设备',
      { type: 'warning', confirmButtonText: '确认撤销' },
    );
    await api.post(`/devices/${device.id}/revoke`);
    ElMessage.success('设备已撤销');
    await load();
  } catch (error) {
    if (error !== 'cancel' && error !== 'close' && !isCanceledRequest(error))
      ElMessage.error(errorMessage(error));
  }
}
function platform(value: string) {
  return platformName(value);
}
function last(value?: string) {
  return formatDateTime(value);
}
onMounted(load);
</script>
<template>
  <div v-loading="loading" class="page">
    <header class="page-header">
      <div>
        <span class="eyebrow">DEVICES</span>
        <h1>设备管理</h1>
        <p>管理连接到此账户的 Agent 安装。</p>
      </div>
    </header>
    <section class="device-grid">
      <article
        v-for="device in devices"
        :key="device.id"
        v-spotlight
        class="panel device-card"
        :class="{ revoked: device.status === 'REVOKED' }"
      >
        <div class="device-card-top">
          <span class="device-icon"
            ><el-icon><Monitor /></el-icon></span
          ><el-tag :type="device.status === 'ACTIVE' ? 'success' : 'info'" effect="plain">{{
            device.status === 'ACTIVE' ? '已连接' : '已撤销'
          }}</el-tag>
        </div>
        <h2>{{ device.deviceName }}</h2>
        <p>{{ platform(device.platform) }} · {{ device.arch }}</p>
        <dl>
          <div>
            <dt>Agent</dt>
            <dd>{{ device.agentVersion || '未知版本' }}</dd>
          </div>
          <div>
            <dt>Device ID</dt>
            <dd>{{ device.deviceId }}</dd>
          </div>
          <div>
            <dt>最近同步</dt>
            <dd>{{ last(device.lastActiveAt) }}</dd>
          </div>
        </dl>
        <div class="device-card-actions">
          <el-button :icon="EditPen" :disabled="device.status !== 'ACTIVE'" @click="rename(device)"
            >重命名</el-button
          ><el-button
            :icon="CircleClose"
            type="danger"
            plain
            :disabled="device.status !== 'ACTIVE'"
            @click="revoke(device)"
            >撤销</el-button
          >
        </div>
      </article>
      <article v-if="!devices.length" v-spotlight class="panel empty-device">
        <el-icon><Monitor /></el-icon>
        <h2>还没有设备</h2>
        <p>在电脑上运行 tokenpulse login 开始连接。</p>
      </article>
    </section>
  </div>
</template>
