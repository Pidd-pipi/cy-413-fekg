import {request} from '../utils/request';
import type {WeeklyReport} from '../types';

// 生成（或取回）14 天情绪周报；同一账号同一结束日重复生成会返回已有快照。
export const generateWeeklyReport=(endDate?:string)=>request<WeeklyReport>('/weekly-reports/generate',{method:'POST',body:JSON.stringify(endDate?{end_date:endDate}:{})});
// 打开当前最新一期已固化周报。
export const getLatestWeeklyReport=()=>request<WeeklyReport>('/weekly-reports/latest');
// 按结束日打开已有周报。
export const getWeeklyReportByEndDate=(endDate:string)=>request<WeeklyReport>(`/weekly-reports?end_date=${endDate}`);
// 按周报编号打开。
export const getWeeklyReport=(id:number)=>request<WeeklyReport>(`/weekly-reports/${id}`);
