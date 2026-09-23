import {useCallback,useState} from 'react';
import {message} from 'antd';
import {generateWeeklyReport,getCurrentWeeklyReport} from '../api/weeklyReport';
import {WEEKLY_REPORT_NOT_FOUND_CODE} from '../constants/weeklyReport';
import {ApiError} from '../utils/request';
import type {WeeklyReport} from '../types';

// useWeeklyReport 负责当前 14 天情绪周报的打开（不存在则生成）、查看与固化语义提示。
export function useWeeklyReport(){
  const [report,setReport]=useState<WeeklyReport|null>(null);
  const [open,setOpen]=useState(false);
  const [loading,setLoading]=useState(false);

  const openReport=useCallback(async(endDate?:string)=>{
    setOpen(true);setLoading(true);
    try{
      const current=await getCurrentWeeklyReport(endDate);
      setReport(current);
    }catch(e){
      if(e instanceof ApiError&&e.code===WEEKLY_REPORT_NOT_FOUND_CODE){
        try{
          const generated=await generateWeeklyReport(endDate);
          setReport(generated);
          message.success('14 天情绪周报已生成并固化');
        }catch(genErr){message.error((genErr as Error).message);setOpen(false)}
      }else{
        message.error((e as Error).message);setOpen(false);
      }
    }finally{setLoading(false)}
  },[]);

  const regenerate=useCallback(async(endDate?:string)=>{
    setLoading(true);
    try{
      const existing=await generateWeeklyReport(endDate);
      setReport(existing);
      if(existing.already_exists){message.info('该结束日的周报已存在，已为你打开已有快照')}
      else{message.success('14 天情绪周报已生成并固化')}
    }catch(e){message.error((e as Error).message)}finally{setLoading(false)}
  },[]);

  const close=useCallback(()=>{setOpen(false)},[]);
  return {report,open,loading,openReport,regenerate,close};
}
