package agent

// turnExecution 一次「用户输入 → runAgentLoop」执行期：组合 turnRuntime、本回合事件通道与内层循环状态。
type turnExecution struct {
	turn *turnRuntime
	out  chan<- AgentEvent
	loop agentLoopState
}
