const {createSnapshot, restoreSnapshot} = require("../helpers/snapshot.js")
const blsData = require("../helpers/data.js")
const initContracts = require('../helpers/initContracts')
const assert = require('chai').assert
const {contract, accounts} = require("@openzeppelin/test-environment")

describe('KeepRandomBeaconService/PricingFees', function() {
    let serviceContract;
    let operatorContract;

    before(async () => {
        let contracts = await initContracts(

        contract.fromArtifact('KeepToken'),
        contract.fromArtifact('TokenStaking'),
        contract.fromArtifact('KeepRandomBeaconService'),
        contract.fromArtifact('KeepRandomBeaconServiceImplV1'),
        contract.fromArtifact('KeepRandomBeaconOperatorPricingStub')
        );
    
        serviceContract = contracts.serviceContract;
        operatorContract = contracts.operatorContract;

        await operatorContract.registerNewGroup(blsData.groupPubKey);
    });

    beforeEach(async () => {
        await createSnapshot()
    });
    
    afterEach(async () => {
      await restoreSnapshot()
    });

    it("should correctly evaluate entry verification fee", async () => {
        await operatorContract.setGasPriceCeiling(200, {from: accounts[0]});
        await operatorContract.setEntryVerificationGasEstimate(12);        

        let fees = await serviceContract.entryFeeBreakdown();
        let entryVerificationFee = fees.entryVerificationFee;

        let expectedEntryVerificationFee = 2400; // 200 * 12
        assert.equal(expectedEntryVerificationFee, entryVerificationFee);
    });

    it("should correctly evaluate DKG contribution fee", async () => {
        await operatorContract.setGasPriceCeiling(1234, {from: accounts[0]});
        await operatorContract.setDkgGasEstimate(13);
        await operatorContract.setGroupSelectionGasEstimate(2);

        let fees = await serviceContract.entryFeeBreakdown();
        let dkgContributionFee = fees.dkgContributionFee;

        let expectedDkgContributionFee = 925; // 1234 * (13+2) * 5% = 925.5
        assert.equal(expectedDkgContributionFee, dkgContributionFee);
    });

    it("should correctly evaluate entry fee estimate", async () => {
        await operatorContract.setGasPriceCeiling(200, {from: accounts[0]});
        await operatorContract.setEntryVerificationGasEstimate(12); 
        await operatorContract.setDkgGasEstimate(14); 
        await operatorContract.setGroupSize(13);
        await operatorContract.setGroupMemberBaseReward(3);
        await operatorContract.setGroupSelectionGasEstimate(2);

        let callbackGas = 7;

        let entryFeeEstimate = await serviceContract.entryFeeEstimate(
            callbackGas
        );

        // entry verification fee = 12 * 200 = 2400
        // dkg contribution fee = (14 + 2) * 200 * 5% = 160
        // group profit fee = 13 * 3 = 39
        // callback fee = (10203 + 7) * 200 = 2042000
        // entry fee = 2400 + 160 + 39 + 2042000 = 2044599
        let expectedEntryFeeEstimate = 2044599;
        assert.equal(expectedEntryFeeEstimate, entryFeeEstimate)
    });
});
