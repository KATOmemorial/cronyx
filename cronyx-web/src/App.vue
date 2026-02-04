<template>
  <div class="app-container">
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <h2 style="margin:0">🚀 CronyX 分布式调度中心</h2>
          <el-button type="primary" @click="dialogVisible = true">
            + 新建任务
          </el-button>
        </div>
      </template>

      <el-table :data="jobs" style="width: 100%" v-loading="loading">
        <el-table-column prop="ID" label="ID" width="60" />
        <el-table-column prop="name" label="任务名称" width="180">
          <template #default="scope">
            <el-tag effect="plain">{{ scope.row.name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cron_expr" label="Cron 表达式" width="150" />
        <el-table-column prop="command" label="Shell 命令" />
        <el-table-column label="下次执行时间" width="200">
          <template #default="scope">
            {{ formatTime(scope.row.next_time) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag v-if="scope.row.status === 1" type="success">运行中</el-tag>
            <el-tag v-else type="info">已停止</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button link type="primary" @click="viewLogs(scope.row.ID)">
              查看日志
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="新建任务" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" placeholder="例如：每日备份" />
        </el-form-item>
        <el-form-item label="Cron表达式">
          <el-input v-model="form.cron_expr" placeholder="*/1 * * * *" />
        </el-form-item>
        <el-form-item label="执行命令">
          <el-input v-model="form.command" placeholder="echo 'Hello'" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitJob">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

// 数据定义
const jobs = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const form = ref({
  name: '',
  cron_expr: '',
  command: '',
  status: 1
})

// 1. 获取任务列表 (走代理 /api/jobs -> localhost:8080/jobs)
const fetchJobs = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/jobs')
    jobs.value = res.data.data
  } catch (err) {
    ElMessage.error('获取任务失败')
  } finally {
    loading.value = false
  }
}

// 2. 提交任务
const submitJob = async () => {
  try {
    await axios.post('/api/job', form.value)
    ElMessage.success('任务创建成功')
    dialogVisible.value = false
    fetchJobs() // 刷新列表
  } catch (err) {
    ElMessage.error('创建失败')
  }
}

// 3. 查看日志 (跳转)
const viewLogs = (id) => {
  // 这里我们暂时简单处理，直接调用后端接口看 JSON
  // 以后可以做一个专门的日志弹窗
  window.open(`http://localhost:8080/job/${id}/logs`, '_blank')
}

// 时间格式化工具
const formatTime = (timestamp) => {
  return new Date(timestamp * 1000).toLocaleString()
}

// 页面加载时自动运行
onMounted(() => {
  fetchJobs()
})
</script>

<style scoped>
.app-container {
  padding: 40px;
  max-width: 1200px;
  margin: 0 auto;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>