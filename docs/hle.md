AI-Agent应用试题
一、背景介绍
“人类终极考试”（HLE）是一项涵盖广泛学科的多模态基准测试，旨在成为同类考试中最终的封闭式学术基准。HLE包含2500道题，涵盖数学、人文科学和自然科学等数十个学科。它由全球超过50个国家500所科研院校的各学科约1000名各领域专家共同开发，题型包括选择题和简答题，适用于自动评分。该测评集难度较大，评测耗时预期较长，适合作为AI Agent能力评估。

官网链接：https://lastexam.ai/

我们从中精挑选了10道高质量，与我们业务紧密相关的题目，覆盖场景包括：网络安全（4道）、密码学（1道）、编程（1道）、具身智能（1道）、人工智能（1道）、数据科学（1道）、机器学习（1道），详情如下：
HLE_text_only_10questions .jsonl

二、数据描述

数据包含7个字段：
"id"：问题编号id
“question”：问题内容
“answer”：标准答案
“answer_type”：答案类型
“rationale”：解题过程分析
“raw_subject”：二级场景类型
“category”：一级场景类型

三、提交内容
参考如下prompt进行推理，模型返回结果保存至 response 字段：
```text
SYSTEM_PROMPT = "Your response should be in the following format:\nExplanation: {your explanation for your answer choice}\nAnswer: {your chosen answer}\nConfidence: {your confidence score between 0% and 100% for your answer}"

messages = [
        {"role": "system", "content": SYSTEM_PROMPT}, 
        {"role": "user", "content": question}
    ]
```
注意：最后需提交：1、解题思路（Agent推理过程）；2、推理代码；3、Agent推理结果的报告

四、评分方法
具体测评方法：https://github.com/centerforaisafety/hle
评估脚本参考：
https://github.com/centerforaisafety/hle/blob/main/hle_eval/run_judge_results.py

测评指标：
准确率（Accuracy）。所有前沿模型在“人类最后的考试”任务上的准确率都很低，这凸显了在缩小当前LLM模型与专家级学术能力在封闭式问题上差距方面存在巨大改进空间。当前榜单上Top1的是Gemini3 Pro，acc仅为38.3%。

校准误差（Calibration Error）。校准误差衡量人工智能系统自信不足或自信过高的程度。为了衡量校准误差，我们要求模型给出答案以及0%到100%的置信度。
仅使用minimax2.1等模型，参考如下prompt进行评估：
```text
JUDGE_PROMPT = """Judge whether the following [response] to [question] is correct or not based on the precise and unambiguous [correct_answer] below.

[question]: {question}

[response]: {response}

Your judgement must be in the format and criteria specified below:

extracted_final_answer: The final exact answer extracted from the [response]. Put the extracted answer as 'None' if there is no exact, final answer to extract from the response.

[correct_answer]: {correct_answer}

reasoning: Explain why the extracted_final_answer is correct or incorrect based on [correct_answer], focusing only on if there are meaningful differences between [correct_answer] and the extracted_final_answer. Do not comment on any background to the problem, do not attempt to solve the problem, do not argue for any answer different than [correct_answer], focus only on whether the answers match.

correct: Answer 'yes' if extracted_final_answer matches the [correct_answer] given above, or is within a small margin of error for numerical problems. Answer 'no' otherwise, i.e. if there if there is any inconsistency, ambiguity, non-equivalency, or if the extracted answer is incorrect.
```
