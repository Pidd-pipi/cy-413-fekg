import {request} from '../utils/request';import type {WeeklyReport} from '../types';
// 打开某结束日（默认今天）已固化的 14 天情绪周报；未生成过返回 404(code=1004)。
export const getCurrentWeeklyReport=(endDate?:string)=>request<WeeklyReport>(`/weekly-reports/current${endDate?`?end_date=${endDate}`:''}`);
// 生成当前周报；同一账号同一结束日重复生成，后端返回已有快照（already_exists=true）且不覆盖。
export const generateWeeklyReport=(endDate?:string)=>request<WeeklyReport>('/weekly-reports/generate',{method:'POST',body:endDate?JSON.stringify({end_date:endDate}):'{}'});
