#!/usr/bin/env bash


CPI_PROJECT=$1
CPI_REGIONS="eu02
eu01"

for region in $CPI_REGIONS; do

echo "cleaning $region for $CPI_PROJECT"



  NETWORKS=$(stackit network list --project-id "$CPI_PROJECT" --region "$region" | jq '.[]| select(.name | contains("cpi")| not).id' -r )

  for network in $NETWORKS; do
    echo "cleaning net id $network"
    NICS=$(stackit network-interface list --region "$region" --project-id "$CPI_PROJECT" --network-id "$network" | jq '.[] | select(.type != "gateway" and .type != "metadata").id' -r)
    for nic in $NICS; do
      stackit network-interface delete --region "$region" --project-id "$CPI_PROJECT" --network-id "$network" "$nic" --assume-yes
    done
    stackit network delete --region "$region" --project-id "$CPI_PROJECT" "$network" --assume-yes
  done

  stackit security-group list --region "$region" --project-id "$CPI_PROJECT" | jq '.[] | select(.name | startswith("test-")).id' -r | xargs -n1 stackit security-group delete --project-id "$CPI_PROJECT" --region "$region" --assume-yes


  stackit public-ip list --project-id "$CPI_PROJECT" --region "$region" --label-selector dynamic_vip | jq .[].id -r | xargs -n1 stackit public-ip delete --project-id "$CPI_PROJECT" --region "$region" --assume-yes

  stackit public-ip list --project-id "$CPI_PROJECT" --region "$region" | jq '.[] | select (.networkInterface == null ).id' -r | xargs -n1 stackit public-ip delete --project-id "$CPI_PROJECT" --region "$region" --assume-yes
done
