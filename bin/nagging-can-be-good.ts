#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { NaggingCanBeGoodStack } from '../lib/nagging-can-be-good-stack';
import {Aspects} from "aws-cdk-lib";
import {AwsSolutionsChecks} from "cdk-nag";
const stackProps = {
    sshPubKey: process.env.SSH_PUB_KEY || ' ',
}
const app = new cdk.App();
Aspects.of(app).add(new AwsSolutionsChecks());
new NaggingCanBeGoodStack(app, 'NaggingCanBeGoodStack', {
    ...stackProps,
    env: {account: process.env.CDK_DEFAULT_ACCOUNT, region: process.env.CDK_DEFAULT_REGION},
});